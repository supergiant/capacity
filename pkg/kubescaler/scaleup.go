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

// filterIgnoringPods
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

// hasMachineFor returns a true or false for whether or not there is
// a machine that suits the resource requirements of the Pod sent.
func hasMachineFor(machineTypes []*provider.MachineType, pod *corev1.Pod) bool {
	cpu, mem := getCPUMem(pod)
	for _, m := range machineTypes {
		if m.CPUResource.Cmp(cpu) >= 0 && m.MemoryResource.Cmp(mem) == 1 {
			return true
		}
	}
	return false
}

// bestMachineFor returns a MachineType that best suits the CPU and
// RAM requirements passed to it.
func bestMachineFor(cpu, mem resource.Quantity, machineTypes []*provider.MachineType) (provider.MachineType, error) {
	if len(machineTypes) == 0 {
		return provider.MachineType{}, ErrNoAllowedMachines
	}
	for _, m := range machineTypes {
		if m.CPUResource.Cmp(cpu) >= 0 && m.MemoryResource.Cmp(mem) == 1 {
			return *m, nil
		}
	}
	return *machineTypes[len(machineTypes)-1], nil
}

// hasCPUMemoryContstraints returns a true or false for whether or
// not the Pod passed to it has resource limits or requests.
func hasCPUMemoryContstraints(pod *corev1.Pod) bool {
	cpu, mem := getCPUMem(pod)
	return cpu.Value() != 0 && mem.Value() != 0
}

// hasController returns a true or false for whether or not the Pod
// passed to it has a controller (like a DaemonSet).
func hasController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil
}

// isNewPod returns a true or false for whether or not the Pod passed
// to it is too new to reschedule.
func isNewPod(pod *corev1.Pod, currentTime time.Time) bool {
	// time should be synced for kubescaler & pod
	return pod.CreationTimestamp.Add(unschedulablePodTimeBuffer).After(currentTime)
}

// hasDaemonSetController returns a true or false for whether or not
// the Pod passed belongs to a DaemonSet.
func hasDaemonSetController(pod *corev1.Pod) bool {
	return metav1.GetControllerOf(pod) != nil && metav1.GetControllerOf(pod).Kind == "DaemonSet"
}

// getCPUMem returns the CPU and RAM for the Containers in the Pods
// passed to it.
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

// getCPUMemTo adds up the CPU and RAM resources of the Containers in
// the Pods passed using the resource quantities passed.
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

// totalCPUMem looks at all of the Pods sent to it and returns
// resource quantities for all of them.
func totalCPUMem(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	var cpu, mem resource.Quantity
	for _, pod := range pods {
		getCPUMemTo(&cpu, &mem, pod)
	}
	return cpu, mem
}

// podNames takes a slice of Pods and returns a slice of only the
// pods' names.
func podNames(pods []*corev1.Pod) []string {
	list := make([]string, len(pods))
	for i := range pods {
		list[i] = pods[i].Name
	}
	return list
}
