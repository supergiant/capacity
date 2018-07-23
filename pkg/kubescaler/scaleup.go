package capacity

import (
	"time"

	"github.com/supergiant/capacity/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Kubescaler) scaleUp(unschedulablePods []*corev1.Pod, readyNodes []*corev1.Node, currentTime time.Time) (bool, error) {
	podsToScale := s.filteredUnschedulablePods(unschedulablePods, readyNodes, currentTime)
	if len(podsToScale) > 0 {
		return false, nil
	}

	// calculate required cpu/mem for unscheduled pods and pick up a machine type
	podsCpu, podsMem := totalCPUMem(podsToScale)
	mtype := bestMachineFor(podsCpu, podsMem, s.config.AllowedMachines)
	err := s.workerManager.Create(mtype.Name)

	return true, err
}

func (s *Kubescaler) filteredUnschedulablePods(pods []*corev1.Pod, readyNodes []*corev1.Node, currentTime time.Time) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		cpu, mem := getCPUMem(pod)

		ignore := isNewPod(pod, currentTime) ||
			// skip standalone pods
			metav1.GetControllerOf(pod) == nil ||
			// skip daemonSet pods
			hasDaemonSetController(pod) ||
			// skip pods without explicit resource requests/limits
			cpu.IsZero() || mem.IsZero() ||
			// skip too large pods
			!hasMachineFor(cpu, mem, s.config.AllowedMachines) ||
			// skip pod if it could be scheduled on one of the ready nodes
			hasEnoughResources(readyNodes, cpu, mem)

		if !ignore {
			filtered = append(filtered, pod)

		}
	}
	return filtered
}

func hasMachineFor(cpu, mem resource.Quantity, machineTypes []provider.MachineType) bool {
	for _, m := range machineTypes {
		if m.CPU.Cmp(cpu) >= 0 && m.Memory.Cmp(mem) == 1 {
			return true
		}
	}
	return false
}

func bestMachineFor(cpu, mem resource.Quantity, machineTypes []provider.MachineType) provider.MachineType {
	for _, m := range machineTypes {
		if m.CPU.Cmp(cpu) >= 0 && m.Memory.Cmp(mem) == 1 {
			return m
		}
	}
	return machineTypes[len(machineTypes)-1]
}

func isNewPod(pod *corev1.Pod, currentTime time.Time) bool {
	return pod.CreationTimestamp.Add(unschedulablePodTimeBuffer).After(currentTime)
}

func hasDaemonSetController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil && metav1.GetControllerOf(pod).Kind == "DaemonSet"
}

func hasEnoughResources(readyNodes []*corev1.Node, cpu, mem resource.Quantity) bool {
	for _, n := range readyNodes {
		if n.Status.Allocatable.Cpu().Cmp(cpu) >= 0 && n.Status.Allocatable.Memory().Cmp(mem) >= 0 {
			return true
		}
	}
	return false
}

func getCPUMem(pod *corev1.Pod) (resource.Quantity, resource.Quantity) {
	var cpu, mem resource.Quantity
	for _, c := range pod.Spec.Containers {
		if !c.Resources.Limits.Cpu().IsZero() {
			cpu.Add(*c.Resources.Limits.Cpu())
		} else {
			cpu.Add(*c.Resources.Requests.Cpu())
		}

		if !c.Resources.Limits.Memory().IsZero() {
			cpu.Add(*c.Resources.Limits.Memory())
		} else {
			cpu.Add(*c.Resources.Requests.Memory())
		}
	}
	return cpu, mem
}

func getCPUMemTo(cpu, mem resource.Quantity, pod *corev1.Pod) {
	for _, c := range pod.Spec.Containers {
		if !c.Resources.Limits.Cpu().IsZero() {
			cpu.Add(*c.Resources.Limits.Cpu())
		} else {
			cpu.Add(*c.Resources.Requests.Cpu())
		}

		if !c.Resources.Limits.Memory().IsZero() {
			cpu.Add(*c.Resources.Limits.Memory())
		} else {
			cpu.Add(*c.Resources.Requests.Memory())
		}
	}
}

func totalCPUMem(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	var cpu, mem resource.Quantity
	for _, pod := range pods {
		getCPUMemTo(cpu, mem, pod)
	}
	return cpu, mem
}
