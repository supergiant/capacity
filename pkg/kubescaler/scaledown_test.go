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
			pods: []*corev1.Pod{&podWithHugeRequests, &podStandAlone},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeRequests, &podWithRequests},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeRequests, &podWithRequests},
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
			providerErr:     errFake,
			expectedErr:     errFake,
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

		err := ks.scaleDown(tc.pods, tc.workerList, nil, time.Now())
		require.Equalf(t, tc.expectedErr, err, "TC#%d", i+1)
	}

}

func TestPodsPerNode(t *testing.T) {
	pods := []*corev1.Pod{&podStandAlone, &podWithRequests}
	require.Equal(t, map[string][]string{"": {podStandAlone.Name}, NodeReadyName: {podWithRequests.Name}}, nodePodsMap(pods))
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
		require.Equalf(t, tc.expectedRes, filterOutStandalonePods(tc.pods), "TC#%d", i+1)
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
		require.Equalf(t, tc.expectedRes, filterOutDaemonSetPods(tc.pods), "TC#%d", i+1)
	}
}

func TestGetEmpty(t *testing.T) {
	tcs := []struct {
		workerList *workers.WorkerList
		nodePods   map[string][]string
		expected   []*workers.Worker
	}{
		{ // TC#1
		},
		{ // TC#2
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			expected: []*workers.Worker{
				{
					NodeName: NodeReadyName,
				},
			},
		},
		{ // TC#3
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {"pod"}},
			expected: []*workers.Worker{},
		},
		{ // TC#4
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {}},
			expected: []*workers.Worker{
				{
					NodeName: NodeReadyName,
				},
			},
		},
		{ // TC#4
			workerList: &workers.WorkerList{
				Items: []*workers.Worker{
					{
						NodeName: NodeReadyName,
					},
					{
						NodeName: NodeScaleDownName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {"pod"}, NodeScaleDownName: {}},
			expected: []*workers.Worker{
				{
					NodeName: NodeScaleDownName,
				},
			},
		},
	}

	for i, tc := range tcs {
		require.Equalf(t, tc.expected, getEmpty(tc.workerList, tc.nodePods), "TC#%d", i+1)
	}
}
