package filters

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// GetReadyNodes filters ready ones from the input node list.
func GetReadyNodes(nodes []*corev1.Node) []*corev1.Node {
	readyNodes := make([]*corev1.Node, 0)
	for _, node := range nodes {
		if IsNodeReadyAndSchedulable(node) {
			readyNodes = append(readyNodes, node)
		}
	}
	return readyNodes
}

// GetScheduledPods filters scheduled ones from the input pod list.
func GetScheduledPods(pods []*corev1.Pod) []*corev1.Pod {
	scheduled := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if IsPodScheduled(pod) {
			scheduled = append(scheduled, pod)
		}
	}
	return scheduled
}

// GetUnschedulablePods filters unschedulable ones from the input pod list.
func GetUnschedulablePods(pods []*corev1.Pod) []*corev1.Pod {
	unschedulable := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if IsPodUnschedulable(pod) {
			unschedulable = append(unschedulable, pod)
		}
	}
	return unschedulable
}

// IsPodScheduled returns true if pod has been scheduled and is in "pending" or "running" phase.
func IsPodScheduled(p *corev1.Pod) bool {
	// TODO: review status of the scheduled pods
	return p.Spec.NodeName != "" && !itemIn(string(p.Status.Phase), []string{string(corev1.PodSucceeded), string(corev1.PodFailed)})
}

// IsPodUnschedulable returns true if pod has not been scheduled and is in "pending", "running" or "unknown" phase.
func IsPodUnschedulable(p *corev1.Pod) bool {
	// TODO: add a check for: reason == "Unschedulable"
	return p.Spec.NodeName == "" && !itemIn(string(p.Status.Phase), []string{string(corev1.PodSucceeded), string(corev1.PodFailed)})
}

func itemIn(item string, list []string) bool {
	for i := range list {
		if list[i] == item {
			return true
		}
	}
	return false
}

// IsNodeReadyAndSchedulable returns true if the node is ready and schedulable.
func IsNodeReadyAndSchedulable(node *corev1.Node) bool {
	ready, _, _ := GetReadinessState(node)
	if !ready {
		return false
	}
	if node.Spec.Unschedulable {
		return false
	}
	return true
}

// GetReadinessState gets readiness state for the node.
// TODO: review it due to https://kubernetes.io/docs/concepts/architecture/nodes/#condition
func GetReadinessState(node *corev1.Node) (isNodeReady bool, lastTransitionTime time.Time, err error) {
	canNodeBeReady, readyFound := true, false
	lastTransitionTime = time.Time{}

	for _, cond := range node.Status.Conditions {
		switch cond.Type {
		case corev1.NodeReady:
			readyFound = true
			if cond.Status == corev1.ConditionFalse || cond.Status == corev1.ConditionUnknown {
				canNodeBeReady = false
			}
			if lastTransitionTime.Before(cond.LastTransitionTime.Time) {
				lastTransitionTime = cond.LastTransitionTime.Time
			}
		case corev1.NodeOutOfDisk:
			if cond.Status == corev1.ConditionTrue {
				canNodeBeReady = false
			}
			if lastTransitionTime.Before(cond.LastTransitionTime.Time) {
				lastTransitionTime = cond.LastTransitionTime.Time
			}
		case corev1.NodeNetworkUnavailable:
			if cond.Status == corev1.ConditionTrue {
				canNodeBeReady = false
			}
			if lastTransitionTime.Before(cond.LastTransitionTime.Time) {
				lastTransitionTime = cond.LastTransitionTime.Time
			}
		}
	}
	if !readyFound {
		return false, time.Time{}, fmt.Errorf("readiness information not found")
	}
	return canNodeBeReady, lastTransitionTime, nil
}
