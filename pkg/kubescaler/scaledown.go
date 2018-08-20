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
func (s *Kubescaler) scaleDown(scheduledPods []*corev1.Pod, workerList *workers.WorkerList, ignoreLabels map[string]string, currentTime time.Time) error {
	// TODO: don't skip failed stateful pods?
	scheduledPods = filterDaemonSetPods(filterStandalonePods(scheduledPods))
	//We are passing nil for masterNodes
	nodePodsMap := nodePodMap(scheduledPods, nil)

	emptyWorkers := getEmpty(workerList, nodePodsMap)
	if len(emptyWorkers) == 0 {
		return nil
	}
	log.Debugf("kubescaler: scale down: nodepods %#v", nodePodsMap)
	log.Debugf("kubescaler: scale down: nodes to delete: %v", workerNodeNames(emptyWorkers))

	removed := make([]string, 0)
	ignored := make([]string, 0)
	defer func() {
		if len(ignored) != 0 {
			log.Debugf("kubescaler: scale down: ignored nodes %v", ignored)
		}
		if len(removed) != 0 {
			log.Infof("kubescaler: scale down: deleted nodes %v", removed)
		}
	}()

	for _, w := range emptyWorkers {
		if reason := ignoreReason(w, ignoreLabels, currentTime); reason != "" {
			ignored = append(ignored, fmt.Sprintf("%s(%s,%s)", w.NodeName, w.MachineID, reason))
			continue
		}

		if _, err := s.DeleteWorker(context.Background(), w.NodeName, w.MachineID); err != nil {
			return err
		}
		removed = append(removed, fmt.Sprintf("%s(%s)", w.NodeName, w.MachineID))
	}

	return nil
}

func ignoreReason(w *workers.Worker, ignoreLabels map[string]string, currentTime time.Time) string {
	switch {
	case w.Reserved:
		return "reserved=true"
	case hasIgnoredLabel(w, ignoreLabels):
		return "ignoredLabel=true"
	case isNewWorker(w, currentTime):
		return "lifespan=" + currentTime.Sub(w.CreationTimestamp).String()
	}
	return ""
}

func nodePodMap(pods []*corev1.Pod, masters []*corev1.Node) map[string]int {
	m := make(map[string]int)
pods:
	for _, pod := range pods {
		//This loop excludes any pods that are running on masters.
		for _, m := range masters {
			if m.Name == pod.Spec.NodeName {
				continue pods
			}
		}
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

func hasIgnoredLabel(worker *workers.Worker, ignored map[string]string) bool {
	if ignored != nil {
		for ignoredK, ignoredV := range ignored {
			val, ok := worker.NodeLabels[ignoredK]
			if !ok {
				continue
			}
			if val == ignoredV {
				return true
			}
		}
	}
	return false
}
