package capacity

import (
	"sync"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
)

func TestKubescalerScaleDown(t *testing.T) {
	tcs := []struct {
		pods            []*corev1.Pod
		workerList      *workers.WorkerList
		allowedMachines []string
		providerErr     error
		expectedErr     error
	}{
		{
			pods: []*corev1.Pod{&podNew, &podStandAlone, &podWithRequests},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeLimits, &podStandAlone},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeLimits, &podWithRequests},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeLimits, &podWithRequests},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
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
			WInterface: fake.NewManager(),
		}

		err := ks.scaleDown(tc.pods, tc.workerList, time.Now())
		require.Equalf(t, tc.expectedErr, err, "TC#%d", i+1)
	}

}

func TestPodsPerNode(t *testing.T) {
	pods := []*corev1.Pod{&podStandAlone, &podWithRequests}
	require.Equal(t, map[string]int{"": 1, NodeReadyName: 1}, nodePodMap(pods))
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
		workerList *workers.WorkerList
		nodePods   map[string]int
		expected   []*corev1.Node
	}{
		{
			expected: make([]*corev1.Node, 0),
		},
		{
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			expected: make([]*corev1.Node, 0),
		},
		{
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string]int{"": 42},
			expected: make([]*corev1.Node, 0),
		},
		{
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string]int{"nodeReady": 42},
			expected: []*corev1.Node{&nodeReady},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expected, getEmpty(tc.workerList, tc.nodePods), "TC#%d", i+1)
	}
}
