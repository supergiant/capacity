package capacity

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"sync"

	"github.com/pborman/uuid"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/provider"
)

func TestKubescalerScaleDown(t *testing.T) {
	tcs := []struct {
		pods            []*corev1.Pod
		nodes           []*corev1.Node
		allowedMachines []provider.MachineType
		providerErr     error
		expectedErr     error
	}{
		{
			pods:            []*corev1.Pod{&podNew, &podStandAlone, &podWithRequests},
			nodes:           []*corev1.Node{&nodeReady},
			allowedMachines: []provider.MachineType{allowedMachine},
		},
		{
			pods:            []*corev1.Pod{&podWithHugeLimits, &podStandAlone},
			nodes:           []*corev1.Node{&nodeReady, &NodeScaleDown},
			allowedMachines: []provider.MachineType{allowedMachine},
		},
		{
			pods:            []*corev1.Pod{&podWithHugeLimits, &podWithRequests},
			nodes:           []*corev1.Node{&nodeReady, &NodeScaleDown},
			allowedMachines: []provider.MachineType{allowedMachine},
		},
		{
			pods:            []*corev1.Pod{&podWithHugeLimits, &podWithRequests},
			nodes:           []*corev1.Node{&nodeReady, &NodeScaleDown},
			allowedMachines: []provider.MachineType{allowedMachine},
			providerErr:     fakeErr,
			expectedErr:     fakeErr,
		},
	}

	for i, tc := range tcs {
		ks := &Kubescaler{
			PersistentConfig: &PersistentConfig{
				filepath: "/tmp/" + uuid.New(),
				mu:       sync.RWMutex{},
				conf: &Config{
					MachineTypes: tc.allowedMachines,
				},
			},
			Manager: &workers.Manager{
				//provider: &fakeProvider{
				//	err: tc.providerErr,
				//},
			},
		}

		err := ks.scaleDown(tc.pods, tc.nodes)
		require.Equalf(t, tc.expectedErr, err, "TC#%d", i+1)
	}

}

func TestPodsPerNode(t *testing.T) {
	pods := []*corev1.Pod{&podStandAlone, &podWithRequests}
	require.Equal(t, map[string]int{"": 1, NodeReadyName: 1}, podsPerNode(pods))
}

func TestFilterStandalonePods(t *testing.T) {
	tcs := []struct {
		pods        []*corev1.Pod
		expectedRes []*corev1.Pod
	}{
		{
			expectedRes: make([]*corev1.Pod, 0),
		},
		{
			pods:        []*corev1.Pod{&podStandAlone, &podWithLimits, &podWithRequests},
			expectedRes: []*corev1.Pod{&podWithLimits, &podWithRequests},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expectedRes, filterStandalonePods(tc.pods), "TC#%d", i+1)
	}
}

func TestFilterDaemonSetPods(t *testing.T) {
	tcs := []struct {
		pods        []*corev1.Pod
		expectedRes []*corev1.Pod
	}{
		{
			expectedRes: make([]*corev1.Pod, 0),
		},
		{
			pods:        []*corev1.Pod{&podDaemonSet, &podWithLimits, &podWithRequests},
			expectedRes: []*corev1.Pod{&podWithLimits, &podWithRequests},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expectedRes, filterDaemonSetPods(tc.pods), "TC#%d", i+1)
	}
}

func TestNodesWithNoPods(t *testing.T) {
	tcs := []struct {
		nodes    []*corev1.Node
		nodePods map[string]int
		expected []*corev1.Node
	}{
		{
			expected: make([]*corev1.Node, 0),
		},
		{
			nodes:    []*corev1.Node{&nodeReady},
			expected: make([]*corev1.Node, 0),
		},
		{
			nodes:    []*corev1.Node{&nodeReady},
			nodePods: map[string]int{"": 42},
			expected: make([]*corev1.Node, 0),
		},
		{
			nodes:    []*corev1.Node{&nodeReady},
			nodePods: map[string]int{"nodeReady": 42},
			expected: []*corev1.Node{&nodeReady},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expected, nodesWithNoPods(tc.nodes, tc.nodePods), "TC#%d", i+1)
	}
}
