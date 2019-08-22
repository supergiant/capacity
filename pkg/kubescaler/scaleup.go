package kubescaler

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/provider"
)

var ErrNoResourcesRequested = errors.New("empty cpu and RAM value")

func (s *Kubescaler) scaleUp(unscheduledPods []*corev1.Pod, machineTypes []*provider.MachineType, machinesLimit int, currentTime time.Time) (bool, error) {
	podsToScale, podsIgnored := filterPods(unscheduledPods, machineTypes, currentTime)
	if len(podsIgnored) > 0 {
		log.Debugf("ignored pods to scale: %v", podsIgnored)
	}
	if len(podsToScale) == 0 {
		return false, nil
	}

	log.Debugf("kubescaler: run: scale up: unscheduled pods: %v", podNames(podsToScale))

	mtypes, err := machinesToScale(podsToScale, machineTypes, machinesLimit)
	if err != nil {
		return false, errors.Wrap(err, "find an appropriate machine type")
	}

	log.Infof("kubescaler: run: scale up: machines to scale: %v", toNames(mtypes))

	for _, mtype := range mtypes {
		worker, err := s.CreateWorker(context.Background(), mtype.Name)
		if err != nil {
			return true, errors.Wrap(err, "create a worker")
		}
		log.Infof("kubescaler: run: scale up: has created a %s worker (%s)", worker.MachineType, worker.MachineID)
	}

	return true, err
}

func machinesToScale(pods []*corev1.Pod, machineTypes []*provider.MachineType, machinesLimit int) ([]provider.MachineType, error) {
	smallBoxesNeeded, err := smallBoxesFor(pods, machineTypes, machinesLimit)
	if err != nil {
		return nil, errors.Wrap(err, "calculate small boxes")
	}
	largeBoxesNeeded, err := largeBoxesFor(pods, machineTypes, machinesLimit)
	if err != nil {
		return nil, errors.Wrap(err, "calculate large box")
	}

	log.Debugf("kubescaler: run: scale up: smallBoxesNeeded: %v, price=%f", toNames(smallBoxesNeeded), priceFor(smallBoxesNeeded))
	log.Debugf("kubescaler: run: scale up: largeBoxNeeded: %s, price=%f", toNames(largeBoxesNeeded), priceFor(largeBoxesNeeded))

	if priceFor(smallBoxesNeeded) <= priceFor(largeBoxesNeeded) {
		return smallBoxesNeeded, nil
	}
	return largeBoxesNeeded, nil
}

func filterPods(pods []*corev1.Pod, allowedMachines []*provider.MachineType, currentTime time.Time) ([]*corev1.Pod, []string) {
	toScale := make([]*corev1.Pod, 0)
	ignored := make([]string, 0)
	for _, pod := range pods {
		ignore, reason := isIgnored(pod, allowedMachines, currentTime)
		if ignore {
			ignored = append(ignored, fmt.Sprintf("%s/%s=%s", pod.Namespace, pod.Name, reason))
			continue
		}
		toScale = append(toScale, pod)
	}
	return toScale, ignored
}

func isIgnored(pod *corev1.Pod, allowedMachines []*provider.MachineType, currentTime time.Time) (bool, string) {
	switch {
	case isNewPod(pod, currentTime):
		return true, "new-pod"
	case !hasController(pod):
		// skip standalone pods
		return true, "standalone-pod"
	case hasDaemonSetController(pod):
		// skip daemonSet pods
		return true, "daemonset-pod"
	case !hasCPUMemoryContstraints(pod):
		// skip pods without explicit resource requests
		return true, "no-requests-set"
	case !hasMachineFor(allowedMachines, pod):
		// skip too large pods
		return true, "pod-exceeds-available-machine-resources"
	}
	return false, ""
}

func hasMachineFor(machineTypes []*provider.MachineType, pod *corev1.Pod) bool {
	cpu, mem := getCPUMemForScheduling(pod)
	for _, m := range machineTypes {
		if hasResources(m, cpu, mem) {
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

	var biggest *provider.MachineType
	// machine types are sorted by price
	for _, m := range provider.SortedMachineTypes(machineTypes) {
		if hasResources(m, cpu, mem) {
			return *m, nil
		}
		biggest = takeBest(biggest, m)
	}
	return *biggest, nil
}

func hasCPUMemoryContstraints(pod *corev1.Pod) bool {
	cpu, mem := getCPUMemForScheduling(pod)
	return cpu.Value() != 0 && mem.Value() != 0
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

func totalCPUMem(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	var cpu, mem resource.Quantity
	for _, pod := range pods {
		pcpu, pmem := getCPUMemForScheduling(pod)
		cpu.Add(pcpu)
		mem.Add(pmem)
	}
	return cpu, mem
}

func smallestCPUMem(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	if len(pods) == 0 {
		return resource.Quantity{}, resource.Quantity{}
	}
	sort.Slice(pods, func(i, j int) bool {
		icpu, imem := getCPUMemForScheduling(pods[i])
		jcpu, jmem := getCPUMemForScheduling(pods[j])
		lessCPU := icpu.Cmp(jcpu) == -1
		equalCPU := icpu.Cmp(jcpu) == 0
		lessMem := imem.Cmp(jmem) == -1
		if equalCPU {
			return lessMem
		}
		return lessCPU
	})
	return getCPUMemForScheduling(pods[0])
}

func smallestMemCPU(pods []*corev1.Pod) (resource.Quantity, resource.Quantity) {
	if len(pods) == 0 {
		return resource.Quantity{}, resource.Quantity{}
	}
	sort.Slice(pods, func(i, j int) bool {
		icpu, imem := getCPUMemForScheduling(pods[i])
		jcpu, jmem := getCPUMemForScheduling(pods[j])
		lessMem := imem.Cmp(jmem) == -1
		equalMem := imem.Cmp(jmem) == 0
		lessCPU := icpu.Cmp(jcpu) == -1
		if equalMem {
			return lessCPU
		}
		return lessMem
	})
	return getCPUMemForScheduling(pods[0])
}

func podNames(pods []*corev1.Pod) []string {
	list := make([]string, len(pods))
	for i := range pods {
		list[i] = fmt.Sprintf("%s/%s", pods[i].Namespace, pods[i].Name)
	}
	return list
}

func hasResources(m *provider.MachineType, cpu, mem resource.Quantity) bool {
	// machine.cpu >= requested.cpu && machine.mem >= requested.mem
	return m.CPUResource.Cmp(cpu) >= 0 && m.MemoryResource.Cmp(mem) >= 0
}

func takeBest(old, new *provider.MachineType) *provider.MachineType {
	if old == nil {
		return new
	}
	if new != nil && new.PriceHour <= old.PriceHour {
		moreCPU := new.CPUResource.Cmp(old.CPUResource) == 1
		equalCPU := new.CPUResource.Cmp(old.CPUResource) == 0
		moreMemory := new.MemoryResource.Cmp(old.MemoryResource) == 1

		if moreCPU || (equalCPU && moreMemory) {
			return new
		}
	}
	return old
}

type machineInfo struct {
	mtype provider.MachineType
	pods  []*corev1.Pod
}

func new(mtype provider.MachineType) machineInfo {
	return machineInfo{
		mtype: mtype,
		pods:  make([]*corev1.Pod, 0),
	}
}

func (mi *machineInfo) resourcesAvailable() (resource.Quantity, resource.Quantity) {
	cpu, mem := mi.mtype.CPUResource, mi.mtype.MemoryResource
	for _, p := range mi.pods {
		pcpu, pmem := getCPUMemForScheduling(p)
		cpu.Sub(pcpu)
		mem.Sub(pmem)
	}
	return cpu, mem
}

func (mi *machineInfo) set(pod *corev1.Pod) error {
	cpu, mem := mi.resourcesAvailable()
	pcpu, pmem := getCPUMemForScheduling(pod)

	if cpu.Cmp(pcpu) == -1 {
		return fmt.Errorf("cpu: available=%s < need=%s", cpu.String(), pcpu.String())
	}
	if mem.Cmp(pmem) == -1 {
		return fmt.Errorf("memory: available=%s < need=%s", mem.String(), pmem.String())
	}

	if mi.pods == nil {
		mi.pods = make([]*corev1.Pod, 0)
	}
	mi.pods = append(mi.pods, pod)
	return nil
}

func largeBoxesFor(pods []*corev1.Pod, machineTypes []*provider.MachineType, machinesLimit int) ([]provider.MachineType, error) {
	podsCopy := make([]*corev1.Pod, 0, len(pods))
	for _, p := range pods {
		podsCopy = append(podsCopy, p)
	}

	machinesNeeded := make([]machineInfo, 0)
	for {
		if len(podsCopy) == 0 {
			break
		}

		// take the largest VM and assign pods to it
		cpu, mem := totalCPUMem(podsCopy)
		mtype, err := bestMachineFor(cpu, mem, machineTypes)
		if err != nil {
			return nil, err
		}

		mi := machineInfo{mtype: mtype}
		unscheduledPods := make([]*corev1.Pod, 0)
		for _, pod := range podsCopy {
			if err := mi.set(pod); err != nil {
				unscheduledPods = append(unscheduledPods, pod)
			}
		}

		machinesNeeded = append(machinesNeeded, mi)
		if len(machinesNeeded) >= machinesLimit {
			break
		}
		podsCopy = unscheduledPods
	}

	return toVMTypes(machinesNeeded), nil
}

func smallBoxesFor(pods []*corev1.Pod, machineTypes []*provider.MachineType, machinesLimit int) ([]provider.MachineType, error) {
	machinesNeeded := make([]machineInfo, 0)
	for _, pod := range sortByCPUMem(pods) {
		if assignTo(machinesNeeded, pod) {
			continue
		}
		mtype, err := bestMachineForPod(pod, machineTypes)
		if err != nil {
			return nil, errors.Wrapf(err, "pod %s/%s", pod.Namespace, pod.Name)
		}

		machinesNeeded = append(machinesNeeded, machineInfo{
			mtype: mtype,
			pods:  []*corev1.Pod{pod},
		})
		if len(machinesNeeded) >= machinesLimit {
			break
		}

	}
	return toVMTypes(machinesNeeded), nil
}

func assignTo(machineInfos []machineInfo, pod *corev1.Pod) bool {
	for _, mi := range machineInfos {
		if err := mi.set(pod); err == nil {
			return true
		}
	}
	return false
}

func sortByCPUMem(pods []*corev1.Pod) []*corev1.Pod {
	if len(pods) == 0 {
		return pods
	}
	sort.Slice(pods, func(i, j int) bool {
		icpu, imem := getCPUMemForScheduling(pods[i])
		jcpu, jmem := getCPUMemForScheduling(pods[j])
		lessCPU := icpu.Cmp(jcpu) == -1
		equalCPU := icpu.Cmp(jcpu) == 0
		lessMem := imem.Cmp(jmem) == -1
		if equalCPU {
			return lessMem
		}
		return lessCPU
	})
	return pods
}

func bestMachineForPod(pod *corev1.Pod, machineTypes []*provider.MachineType) (provider.MachineType, error) {
	pcpu, pmem := getCPUMemForScheduling(pod)
	return bestMachineFor(pcpu, pmem, machineTypes)
}

func toVMTypes(machineInfos []machineInfo) []provider.MachineType {
	out := make([]provider.MachineType, 0, len(machineInfos))
	for _, mi := range machineInfos {
		out = append(out, mi.mtype)
	}
	return out
}

func toNames(mtypes []provider.MachineType) []string {
	out := make([]string, 0, len(mtypes))
	for _, mtype := range mtypes {
		out = append(out, mtype.Name)
	}
	return out
}

func priceFor(mtypes []provider.MachineType) float64 {
	var out float64
	for _, mtype := range mtypes {
		out += mtype.PriceHour
	}
	return out
}
