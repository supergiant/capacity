package capacity

import (
	"sync"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
	"github.com/supergiant/capacity/pkg/provider"
)

var (
	currentTime = time.Now()
	trueVar     = true
	errFake     = errors.New("fake error")

	resource13   = resource.MustParse("13")
	resource13Mi = resource.MustParse("13Mi")
	resource33   = resource.MustParse("33")
	resource33Mi = resource.MustParse("33Mi")
	resource42   = resource.MustParse("42")
	resource42Mi = resource.MustParse("42Mi")

	machineType13 = provider.MachineType{Name: "13", CPUResource: resource13, MemoryResource: resource13Mi}
	machineType42 = provider.MachineType{Name: "42", CPUResource: resource42, MemoryResource: resource42Mi}

	resourceList13CPU13Mi = corev1.ResourceList{
		"cpu":    resource13,
		"memory": resource13Mi,
	}
	resourceList33CPU33Mi = corev1.ResourceList{
		"cpu":    resource33,
		"memory": resource33Mi,
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
			Allocatable: resourceList13CPU13Mi,
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
						Requests: resourceList13CPU13Mi,
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
						Limits: resourceList33CPU33Mi,
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
		ks := &Kubescaler{
			PersistentConfig: &PersistentConfig{
				filepath: "/tmp/" + uuid.New(),
				mu:       sync.RWMutex{},
				conf: &Config{
					MachineTypes: tc.allowedMachines,
				},
			},
			WInterface: fake.NewManager(tc.providerErr),
		}

		_, err := ks.scaleUp(tc.pods, allowedMachines, currentTime)
		require.Equalf(t, tc.expectedErr, errors.Cause(err), "TC#%d", i+1)
	}

}

func TestFilterIgnoringPos(t *testing.T) {
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

	res := filterIgnoringPods(pods, allowedMachines, currentTime)
	require.Equal(t, expectedRes, res)
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
			machineTypes: []*provider.MachineType{&machineType42},
			expectedRes:  true,
		},
	}

	for i, tc := range tcs {
		res := hasMachineFor(tc.machineTypes, tc.pod)
		require.Equalf(t, tc.expectedRes, res, "TC#%d", i+1)
	}
}

func TestBestMachineFor(t *testing.T) {
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
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedErr:  ErrNoResourcesRequested,
		},
		{ // TC#3
			cpu:          resource.MustParse("1"),
			mem:          resource.MustParse("1Mi"),
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedRes:  machineType13,
		},
		{ // TC#4
			cpu:          resource.MustParse("13"),
			mem:          resource.MustParse("12Mi"),
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedRes:  machineType13,
		},
		{ // TC#5
			cpu:          resource.MustParse("13"),
			mem:          resource.MustParse("13Mi"),
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedRes:  machineType42,
		},
		{ // TC#6
			cpu:          resource.MustParse("35"),
			mem:          resource.MustParse("45Mi"),
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedRes:  machineType42,
		},
		{ // TC#7
			cpu:          resource.MustParse("64"),
			mem:          resource.MustParse("64Mi"),
			machineTypes: []*provider.MachineType{&machineType13, &machineType42},
			expectedRes:  machineType42,
		},
		{ // TC#8
			cpu:          resource.MustParse("64"),
			mem:          resource.MustParse("64Mi"),
			machineTypes: []*provider.MachineType{&machineType42, &machineType13},
			expectedRes:  machineType42,
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
		{&podWithRequests, resource13, resource13Mi},
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
	expectedCPU, expectedMem := resource.MustParse("46"), resource.MustParse("46Mi")

	cpu, mem := totalCPUMem(pods)
	require.Equal(t, expectedCPU.Value(), cpu.Value(), "cpu")
	require.Equal(t, expectedMem.Value(), mem.Value(), "mem")
}
