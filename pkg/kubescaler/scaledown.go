package capacity

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
)

func (s *Kubescaler) scaleDown(ctx context.Context, scheduledPods []*corev1.Pod, readyNodes []*corev1.Node) error {
	scheduledPods = filterDaemonSetPods(filterStandalonePods(scheduledPods))
	nodeMap := podsPerNode(scheduledPods)

	emptyNodes := nodesWithNoPods(readyNodes, nodeMap)
	if len(emptyNodes) == 0 {
		return nil
	}

	// TODO: log empty nodes
	for _, node := range emptyNodes {
		if !workers.IsReserved(node) {
			// gracefully remove a node
			if _, err := s.DeleteWorker(ctx, node.Name, node.Spec.ProviderID); err != nil {
				return err
			}
		}
	}

	return nil
}

func podsPerNode(pods []*corev1.Pod) map[string]int {
	m := make(map[string]int)
	for _, pod := range pods {
		m[pod.Spec.NodeName]++
	}
	return m
}

func filterStandalonePods(pods []*corev1.Pod) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if v1.GetControllerOf(pod) != nil {
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

func nodesWithNoPods(readyNodes []*corev1.Node, nodePods map[string]int) []*corev1.Node {
	nonEmptyNodes := sets.NewString()
	for nodeName, _ := range nodePods {
		nonEmptyNodes.Insert(nodeName)
	}
	emptyNodes := make([]*corev1.Node, 0)
	for _, node := range readyNodes {
		if nonEmptyNodes.Has(node.Name) {
			emptyNodes = append(emptyNodes, node)
		}
	}
	return emptyNodes
}
