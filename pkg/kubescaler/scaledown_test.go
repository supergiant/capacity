package kubescaler

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
	"github.com/supergiant/capacity/pkg/persistentfile/file"
)

func TestKubescalerScaleDown(t *testing.T) {
	tcs := []struct {
		pods            []*corev1.Pod
		workerList      *api.WorkerList
		allowedMachines []string
		providerErr     error
		expectedErr     error
	}{
		{
			pods: []*corev1.Pod{&podNew, &podStandAlone, &podWithRequests},
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{NodeName: NodeReadyName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeRequests, &podStandAlone},
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeRequests, &podWithRequests},
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{NodeName: NodeReadyName},
					{NodeName: NodeScaleDownName},
				},
			},
			allowedMachines: []string{allowedMachine.Name},
		},
		{
			pods: []*corev1.Pod{&podWithHugeRequests, &podWithRequests},
			workerList: &api.WorkerList{
				Items: []*api.Worker{
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
		f, err := file.New("/tmp/"+uuid.New(), os.FileMode(0664))
		require.Nilf(t, err, "TC#%d", i+1)

		ks := &Kubescaler{
			configManager: &configManager{
				file: f,
				mu:   sync.RWMutex{},
				conf: api.Config{
					MachineTypes: tc.allowedMachines,
				},
			},
			WInterface: fake.NewManager(tc.providerErr),
		}

		err = ks.scaleDown(tc.pods, tc.workerList, nil, time.Now())
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
		workerList *api.WorkerList
		nodePods   map[string][]string
		expected   []*api.Worker
	}{
		{ // TC#1
		},
		{ // TC#2
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			expected: []*api.Worker{
				{
					NodeName: NodeReadyName,
				},
			},
		},
		{ // TC#3
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {"pod"}},
			expected: []*api.Worker{},
		},
		{ // TC#4
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{
						NodeName: NodeReadyName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {}},
			expected: []*api.Worker{
				{
					NodeName: NodeReadyName,
				},
			},
		},
		{ // TC#4
			workerList: &api.WorkerList{
				Items: []*api.Worker{
					{
						NodeName: NodeReadyName,
					},
					{
						NodeName: NodeScaleDownName,
					},
				},
			},
			nodePods: map[string][]string{NodeReadyName: {"pod"}, NodeScaleDownName: {}},
			expected: []*api.Worker{
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
