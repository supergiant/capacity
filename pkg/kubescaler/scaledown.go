package kubescaler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/log"
)

// TODO: use workers here
func (s *Kubescaler) scaleDown(scheduledPods []*corev1.Pod, workerList *workers.WorkerList, ignoreLabels map[string]string, currentTime time.Time) error {
	// TODO: don't skip failed stateful pods?
	scheduledPods = filterOutDaemonSetPods(filterOutStandalonePods(scheduledPods))
	nodePodsMap := nodePodsMap(scheduledPods)

	emptyWorkers := getEmpty(workerList, nodePodsMap)
	if len(emptyWorkers) == 0 {
		return nil
	}
	log.Debugf("kubescaler: scale down: nodepods %v", nodePodsMap)
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

func nodePodsMap(pods []*corev1.Pod) map[string][]string {
	m := make(map[string][]string)
	for _, pod := range pods {
		if m[pod.Spec.NodeName] == nil {
			m[pod.Spec.NodeName] = []string{pod.Name}
			continue
		}
		m[pod.Spec.NodeName] = append(m[pod.Spec.NodeName], pod.Name)
	}
	return m
}

func filterOutStandalonePods(pods []*corev1.Pod) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if metav1.GetControllerOf(pod) != nil {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func filterOutDaemonSetPods(pods []*corev1.Pod) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if !hasDaemonSetController(pod) {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func getEmpty(workerList *workers.WorkerList, nodePods map[string][]string) []*workers.Worker {
	if workerList == nil || len(workerList.Items) == 0 {
		return nil
	}
	if len(nodePods) == 0 {
		return workerList.Items
	}

	emptyWorkers := make([]*workers.Worker, 0)
	for _, worker := range workerList.Items {
		if worker.NodeName == "" || len(nodePods[worker.NodeName]) > 0 {
			continue
		}
		emptyWorkers = append(emptyWorkers, worker)
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
