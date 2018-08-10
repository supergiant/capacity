package capacity

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/log"
)

// TODO: use workers here
func (s *Kubescaler) scaleDown(scheduledPods []*corev1.Pod, workerList *workers.WorkerList, currentTime time.Time) error {
	scheduledPods = filterDaemonSetPods(filterStandalonePods(scheduledPods))
	nodePodsMap := nodePodMap(scheduledPods)

	emptyWorkers := getEmpty(workerList, nodePodsMap)
	if len(emptyWorkers) == 0 {
		return nil
	}
	log.Debugf("kubescaler: run: scaledown: nodepods %#v", nodePodsMap)
	log.Debugf("kubescaler: run: scale down: nodes to delete: %v", workerNodeNames(emptyWorkers))

	removed := make([]string, 0)
	ignored := make([]string, 0)
	defer func() {
		log.Infof("kubescaler: run: scale down: has deleted %v nodes, %v ignored", removed, ignored)
	}()

	for _, worker := range emptyWorkers {
		if worker.Reserved || isNewWorker(worker, currentTime) {
			ignored = append(ignored, fmt.Sprintf("%s(%s,reserved=%t,lifespan=%s)", worker.NodeName, worker.MachineID,
				worker.Reserved, currentTime.Sub(worker.CreationTimestamp)))
			continue
		}
		if _, err := s.DeleteWorker(context.Background(), worker.NodeName, worker.MachineID); err != nil {
			return err
		}
		removed = append(removed, fmt.Sprintf("%s(%s)", worker.NodeName, worker.MachineID))
	}

	return nil
}

func nodePodMap(pods []*corev1.Pod) map[string]int {
	m := make(map[string]int)
	for _, pod := range pods {
		m[pod.Spec.NodeName]++
	}
	return m
}

func filterStandalonePods(pods []*corev1.Pod) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if metav1.GetControllerOf(pod) != nil {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func filterDaemonSetPods(pods []*corev1.Pod) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if !hasDaemonSetController(pod) {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func getEmpty(workerList *workers.WorkerList, nodePods map[string]int) []*workers.Worker {
	if workerList == nil {
		return nil
	}

	nonEmptyNodes := sets.NewString()
	for nodeName := range nodePods {
		nonEmptyNodes.Insert(nodeName)
	}
	emptyWorkers := make([]*workers.Worker, 0)
	for _, worker := range workerList.Items {
		if worker.NodeName != "" && !nonEmptyNodes.Has(worker.NodeName) {
			emptyWorkers = append(emptyWorkers, worker)
		}
	}
	return emptyWorkers
}

func workerNodeNames(wkrs []*workers.Worker) []string {
	list := make([]string, 0, len(wkrs))
	for _, w := range wkrs {
		if w.NodeName != "" {
			list = append(list, w.NodeName)
		}
	}
	return list
}

func isNewWorker(worker *workers.Worker, currentTime time.Time) bool {
	return worker.CreationTimestamp.Add(workers.MinWorkerLifespan).After(currentTime)
}
