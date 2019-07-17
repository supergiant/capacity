package kubescaler

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
	"github.com/supergiant/capacity/pkg/persistentfile/file"
	"github.com/supergiant/capacity/pkg/provider"
)

var (
	currentTime = time.Now()
	trueVar     = true
	errFake     = errors.New("fake error")

	resource1    = resource.MustParse("1")
	resource1Mi  = resource.MustParse("1Mi")
	resource2    = resource.MustParse("2")
	resource2Mi  = resource.MustParse("2Mi")
	resource42   = resource.MustParse("42")
	resource42Mi = resource.MustParse("42Mi")

	vmPrice1CPU1Mem1 = provider.MachineType{
		Name:           "Price1CPU1Mem1",
		PriceHour:      1,
		CPUResource:    resource1,
		MemoryResource: resource1Mi,
	}
	vmPrice1CPU2Mem1 = provider.MachineType{
		Name:           "Price1CPU2Mem1",
		PriceHour:      1,
		CPUResource:    resource2,
		MemoryResource: resource1Mi,
	}
	vmPrice1CPU1Mem2 = provider.MachineType{
		Name:           "Price1CPU1Mem2",
		PriceHour:      1,
		CPUResource:    resource1,
		MemoryResource: resource2Mi,
	}
	vmPrice1CPU2Mem2 = provider.MachineType{
		Name:           "Price1CPU2Mem2",
		PriceHour:      1,
		CPUResource:    resource2,
		MemoryResource: resource2Mi,
	}
	vmPrice2CPU2Mem2 = provider.MachineType{
		Name:           "Price2CPU2Mem2",
		PriceHour:      2,
		CPUResource:    resource2,
		MemoryResource: resource2Mi,
	}
	machinePrice42Type42 = provider.MachineType{
		Name:           "42",
		CPUResource:    resource42,
		MemoryResource: resource42Mi,
		PriceHour:      42,
	}

	resourceList1CPU1Mi = corev1.ResourceList{
		"cpu":    resource1,
		"memory": resource1,
	}
	resourceList2CPU2Mi = corev1.ResourceList{
		"cpu":    resource2,
		"memory": resource2,
	}
	resourceList42CPU42Mi = corev1.ResourceList{
		"cpu":    resource42,
		"memory": resource42Mi,
	}
	resourceListHuge = corev1.ResourceList{
		"cpu":    resource.MustParse("1024"),
		"memory": resource.MustParse("1024Gi"),
	}
)

var (
	allowedMachine = provider.MachineType{Name: "42cpu42Mi", CPUResource: resource42, MemoryResource: resource42Mi}

	NodeReadyName = "nodeReady"
	nodeReady     = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: NodeReadyName,
		},
		Status: corev1.NodeStatus{
			Allocatable: resourceList1CPU1Mi,
		},
	}
	NodeScaleDownName = "nodeScaleDown"
	NodeScaleDown     = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: NodeScaleDownName,
		},
		Status: corev1.NodeStatus{
			Allocatable: resourceList42CPU42Mi,
		},
	}

	podNew = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podNew",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(time.Hour)},
		},
	}
	podStandAlone = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podStandAlone",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(-time.Hour)},
		},
	}
	podDaemonSet = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podDaemonSet",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(-time.Hour)},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "DaemonSet",
					Controller: &trueVar,
				},
			},
		},
	}
	podWithRequests = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podWithRequests",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(-time.Hour)},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "Deployment",
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: NodeReadyName,
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: resourceList1CPU1Mi,
					},
				},
			},
		},
	}
	podWithLimits = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podWithLimits",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(-time.Hour)},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "Deployment",
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: NodeReadyName,
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Limits: resourceList2CPU2Mi,
					},
				},
			},
		},
	}
	podWithHugeRequests = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "podWithHugeRequests",
			CreationTimestamp: metav1.Time{Time: currentTime.Add(-time.Hour)},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "Deployment",
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: resourceListHuge,
					},
				},
			},
		},
	}
)

type fakeProvider struct {
	provider.Provider
	err error
}

func (p *fakeProvider) CreateMachine(typeName string) (*provider.Machine, error) {
	return nil, p.err
}

func (p *fakeProvider) DeleteMachine(name string) error {
	return p.err
}

func TestKubescalerScaleUp(t *testing.T) {
	tcs := []struct {
		pods            []*corev1.Pod
		nodes           []*corev1.Node
		allowedMachines []string
		providerErr     error
		expectedErr     error
	}{
		{
			nodes:           []*corev1.Node{&nodeReady},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods:            []*corev1.Pod{&podNew, &podStandAlone, &podWithRequests},
			nodes:           []*corev1.Node{&nodeReady},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods:            []*corev1.Pod{&podNew, &podStandAlone, &podWithRequests},
			nodes:           []*corev1.Node{&nodeReady},
			allowedMachines: []string{allowedMachine.Name},
			providerErr:     errFake,
			expectedErr:     errFake,
		},
	}

	allowedMachines := []*provider.MachineType{&allowedMachine}
	for i, tc := range tcs {
		f, err := file.New("/tmp/"+uuid.New(), os.FileMode(0664))
		require.Nilf(t, err, "TC#%d", i+1)

		ks := &Kubescaler{
			configManager: &ConfigManager{
				file: f,
				mu:   sync.RWMutex{},
				conf: api.Config{
					MachineTypes: tc.allowedMachines,
				},
			},
			workerManager: fake.NewManager(tc.providerErr),
		}

		_, err = ks.scaleUp(tc.pods, allowedMachines, currentTime)
		require.Equalf(t, tc.expectedErr, errors.Cause(err), "TC#%d", i+1)
	}

}

func TestFilterPods(t *testing.T) {
	pods := []*corev1.Pod{
		&podNew,
		&podStandAlone,
		&podDaemonSet,
		&podWithLimits,
		&podWithRequests,
		&podWithHugeRequests,
	}
	allowedMachines := []*provider.MachineType{&allowedMachine}
	expectedRes := []*corev1.Pod{&podWithRequests}

	toScale, _ := filterPods(pods, allowedMachines, currentTime)
	require.Equal(t, expectedRes, toScale)
}

func TestHasMachineFor(t *testing.T) {
	tcs := []struct {
		pod          *corev1.Pod
		machineTypes []*provider.MachineType
		expectedRes  bool
	}{
		{
			pod:          &podWithLimits,
			machineTypes: []*provider.MachineType{},
		},
		{
			pod:          &podWithRequests,
			machineTypes: []*provider.MachineType{&vmPrice1CPU1Mem2},
			expectedRes:  true,
		},
	}

	for i, tc := range tcs {
		res := hasMachineFor(tc.machineTypes, tc.pod)
		require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
	}
}

func TestBestMachineFor(t *testing.T) {
	vmTypes := []*provider.MachineType{
		&vmPrice1CPU1Mem1,
		&vmPrice1CPU1Mem2,
		&vmPrice1CPU2Mem1,
		&vmPrice1CPU2Mem2,
		&vmPrice2CPU2Mem2,
		&machinePrice42Type42,
	}

	tcs := []struct {
		cpu, mem     resource.Quantity
		machineTypes []*provider.MachineType
		expectedRes  provider.MachineType
		expectedErr  error
	}{
		{ // TC#1
			cpu:         resource.MustParse("1"),
			mem:         resource.MustParse("1Mi"),
			expectedErr: ErrNoAllowedMachines,
		},
		{ // TC#2
			machineTypes: vmTypes,
			expectedErr:  ErrNoResourcesRequested,
		},
		{ // TC#3
			cpu:          resource.MustParse("1"),
			mem:          resource.MustParse("1Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
		{ // TC#4
			cpu:          resource.MustParse("1"),
			mem:          resource.MustParse("2Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
		{ // TC#5
			cpu:          resource.MustParse("2"),
			mem:          resource.MustParse("1Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
		{ // TC#6
			cpu:          resource.MustParse("2"),
			mem:          resource.MustParse("2Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
		{ // TC#7
			cpu:          resource.MustParse("1"),
			mem:          resource.MustParse("64Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
		{ // TC#9
			cpu:          resource.MustParse("64"),
			mem:          resource.MustParse("64Mi"),
			machineTypes: vmTypes,
			expectedRes:  vmPrice1CPU2Mem2,
		},
	}

	for i, tc := range tcs {
		res, err := bestMachineFor(tc.cpu, tc.mem, tc.machineTypes)
		require.Equalf(t, tc.expectedErr, err, "TC#%d", i+1)
		if err == nil {
			require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
		}
	}
}

func TestIsNewPod(t *testing.T) {
	tcs := []struct {
		pod         *corev1.Pod
		expectedRes bool
	}{
		{&podNew, true},
		{&podStandAlone, false},
		{&podDaemonSet, false},
		{&podWithRequests, false},
	}

	for i, tc := range tcs {
		res := isNewPod(tc.pod, currentTime)
		require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
	}
}

func TestHasController(t *testing.T) {
	tcs := []struct {
		pod         *corev1.Pod
		expectedRes bool
	}{
		{&podDaemonSet, true},
		{&podNew, false},
		{&podStandAlone, false},
		{&podWithRequests, true},
	}

	for i, tc := range tcs {
		res := hasController(tc.pod)
		require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
	}
}

func TestHasDaemonSetController(t *testing.T) {
	tcs := []struct {
		pod         *corev1.Pod
		expectedRes bool
	}{
		{&podDaemonSet, true},
		{&podNew, false},
		{&podStandAlone, false},
		{&podWithRequests, false},
	}

	for i, tc := range tcs {
		res := hasDaemonSetController(tc.pod)
		require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
	}
}

func TestGetCPUMem(t *testing.T) {
	tcs := []struct {
		pod         *corev1.Pod
		expectedCPU resource.Quantity
		expectedMem resource.Quantity
	}{
		{&podNew, resource.Quantity{}, resource.Quantity{}},
		{&podStandAlone, resource.Quantity{}, resource.Quantity{}},
		{&podDaemonSet, resource.Quantity{}, resource.Quantity{}},
		{&podWithRequests, resource1, resource1},
		{&podWithLimits, resource.Quantity{}, resource.Quantity{}},
		{&podWithHugeRequests, resource.MustParse("1024"), resource.MustParse("1024Gi")},
	}

	for i, tc := range tcs {
		cpu, mem := getCPUMemForScheduling(tc.pod)
		require.Equalf(t, tc.expectedCPU.Value(), cpu.Value(), "TC#%d: cpu", i+1)
		require.Equalf(t, tc.expectedMem.Value(), mem.Value(), "TC#%d: mem", i+1)
	}
}

func TestTotalCPUMem(t *testing.T) {
	pods := []*corev1.Pod{
		&podWithRequests,
		&podWithLimits,
	}
	expectedCPU, expectedMem := resource1, resource1

	cpu, mem := totalCPUMem(pods)
	require.Equal(t, expectedCPU.Value(), cpu.Value(), "cpu")
	require.Equal(t, expectedMem.Value(), mem.Value(), "mem")
}
