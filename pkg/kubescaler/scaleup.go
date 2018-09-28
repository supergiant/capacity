package capacity

import (
	"context"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/provider"
)

var ErrNoResourcesRequested = errors.New("empty cpu and RAM value")

func (s *Kubescaler) scaleUp(unscheduledPods []*corev1.Pod, machineTypes []*provider.MachineType, currentTime time.Time) (bool, error) {
	podsToScale := filterIgnoringPods(unscheduledPods, machineTypes, currentTime)
	if len(podsToScale) == 0 {
		return false, nil
	}

	log.Debugf("kubescaler: run: scale up: unscheduled pods: %v", podNames(podsToScale))

	// get required cpu/mem for unscheduled pods and pick up a machine type
	podsCPU, podsMem := totalCPUMem(podsToScale)
	mtype, err := bestMachineFor(podsCPU, podsMem, machineTypes)
	if err != nil {
		return false, errors.Wrap(err, "find an appropriate machine type")
	}

	log.Debugf("kubescaler: run: scale up: unscheduled pods needs cpu=%s, mem=%s: pick the %s machine (cpu=%s, mem=%s)",
		podsCPU.String(), podsMem.String(), mtype.Name, mtype.CPU, mtype.Memory)

	worker, err := s.CreateWorker(context.Background(), mtype.Name)
	if err != nil {
		return true, errors.Wrap(err, "create a worker")
	}

	log.Infof("kubescaler: run: scale up: has created a %s worker (%s)", worker.MachineType, worker.MachineID)
	return true, err
}

func filterIgnoringPods(pods []*corev1.Pod, allowedMachines []*provider.MachineType, currentTime time.Time) []*corev1.Pod {
	filtered := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		ignore := isNewPod(pod, currentTime) ||
			// skip standalone pods
			!hasController(pod) ||
			// skip daemonSet pods
			hasDaemonSetController(pod) ||
			// skip pods without explicit resource requests/limits
			!hasCPUMemoryContstraints(pod) ||
			// skip too large pods
			!hasMachineFor(allowedMachines, pod)

		if !ignore {
			filtered = append(filtered, pod)

		}
	}
	return filtered
}

func hasMachineFor(machineTypes []*provider.MachineType, pod *corev1.Pod) bool {
	cpu, mem := getCPUMemForScheduling(pod)
	for _, m := range machineTypes {
		if m.CPUResource.Cmp(cpu) >= 0 && m.MemoryResource.Cmp(mem) == 1 {
			return true
		}
	}
	return false
}

func bestMachineFor(cpu, mem resource.Quantity, machineTypes []*provider.MachineType) (provider.MachineType, error) {
	if cpu.Value() == 0 && mem.Value() == 0 {
		return provider.MachineType{}, ErrNoResourcesRequested
	}
	if len(machineTypes) == 0 {
		return provider.MachineType{}, ErrNoAllowedMachines
	}

	var biggest provider.MachineType
	for _, m := range provider.SortedMachineTypes(machineTypes) {
		if m.CPUResource.Cmp(cpu) > -1 && m.MemoryResource.Cmp(mem) == 1 {
			return *m, nil
		}
		biggest = *m
	}
	return biggest, nil
}

func hasCPUMemoryContstraints(pod *corev1.Pod) bool {
	cpu, mem := getCPUMemForScheduling(pod)
	return cpu.Value() != 0 || mem.Value() != 0
}

func hasController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil
}

func isNewPod(pod *corev1.Pod, currentTime time.Time) bool {
	// time should be synced for kubescaler & pod
	return pod.CreationTimestamp.Add(unschedulablePodTimeBuffer).After(currentTime)
}

func hasDaemonSetController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil && metav1.GetControllerOf(pod).Kind == "DaemonSet"
}

func getCPUMemForScheduling(pod *corev1.Pod) (resource.Quantity, resource.Quantity) {
	// Scheduling is based on requests.
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/resource-qos.md#requests-and-limits
	var cpu, mem resource.Quantity
	for _, c := range pod.Spec.Containers {
		cpu.Add(*c.Resources.Requests.Cpu())
		mem.Add(*c.Resources.Requests.Memory())
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

func podNames(pods []*corev1.Pod) []string {
	list := make([]string, len(pods))
	for i := range pods {
		list[i] = pods[i].Name
	}
	return list
}
