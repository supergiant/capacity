package capacity

import (
	"context"
	"time"

	"github.com/supergiant/capacity/pkg/providers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Kubescaler) scaleUp(ctx context.Context, unschedulablePods []*corev1.Pod, readyNodes []*corev1.Node, machineTypes []*providers.MachineType, currentTime time.Time) error {
	podsToScale := filterIgnoringPods(unschedulablePods, readyNodes, machineTypes, currentTime)
	if len(podsToScale) == 0 {
		return nil
	}

	// calculate required cpu/mem for unscheduled pods and pick up a machine type
	podsCpu, podsMem := totalCPUMem(podsToScale)
	mtype, err := bestMachineFor(podsCpu, podsMem, machineTypes)
	if err != nil {
		return err
	}

	return s.CreateWorker(ctx, mtype.Name)
}

func filterIgnoringPods(pods []*corev1.Pod, readyNodes []*corev1.Node, allowedMachines []*providers.MachineType, currentTime time.Time) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		cpu, mem := getCPUMem(pod)

		ignore := isNewPod(pod, currentTime) ||
			// skip standalone pods
			!hasController(pod) ||
			// skip daemonSet pods
			hasDaemonSetController(pod) ||
			// skip pods without explicit resource requests/limits
			cpu.Value() == 0 || mem.Value() == 0 ||
			// skip too large pods
			!hasMachineFor(cpu, mem, allowedMachines) ||
			// skip pod if it could be scheduled on one of the ready nodes
			hasEnoughResources(readyNodes, cpu, mem)

		if !ignore {
			filtered = append(filtered, pod)

		}
	}
	return filtered
}

func hasMachineFor(cpu, mem resource.Quantity, machineTypes []*providers.MachineType) bool {
	for _, m := range machineTypes {
		if m.CPU.Cmp(cpu) >= 0 && m.Memory.Cmp(mem) == 1 {
			return true
		}
	}
	return false
}

func bestMachineFor(cpu, mem resource.Quantity, machineTypes []*providers.MachineType) (providers.MachineType, error) {
	if len(machineTypes) == 0 {
		return providers.MachineType{}, ErrNoAllowedMachined
	}
	for _, m := range machineTypes {
		if m.CPU.Cmp(cpu) >= 0 && m.Memory.Cmp(mem) == 1 {
			return *m, nil
		}
	}
	return *machineTypes[len(machineTypes)-1], nil
}

func hasController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil
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
			mem.Add(*c.Resources.Limits.Memory())
		} else {
			mem.Add(*c.Resources.Requests.Memory())
		}
	}
	return cpu, mem
}

func getCPUMemTo(cpu, mem *resource.Quantity, pod *corev1.Pod) {
	for _, c := range pod.Spec.Containers {
		if !c.Resources.Limits.Cpu().IsZero() {
			cpu.Add(*c.Resources.Limits.Cpu())
		} else {
			cpu.Add(*c.Resources.Requests.Cpu())
		}

		if !c.Resources.Limits.Memory().IsZero() {
			mem.Add(*c.Resources.Limits.Memory())
		} else {
			mem.Add(*c.Resources.Requests.Memory())
		}
	}
}

func totalCPUMem(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	var cpu, mem resource.Quantity
	for _, pod := range pods {
		getCPUMemTo(&cpu, &mem, pod)
	}
	return cpu, mem
}
