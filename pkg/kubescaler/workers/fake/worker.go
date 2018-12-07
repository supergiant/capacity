package fake

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/provider"
)

var _ workers.WInterface = &Manager{}

type Manager struct {
	clusterName string
	err         error
}

func NewManager(err error) *Manager {
	return &Manager{
		clusterName: "fake",
		err:         err,
	}
}

func (m *Manager) MachineTypes() []*provider.MachineType {
	return []*provider.MachineType{
		{
			Name:           "m4.large",
			CPUResource:    resource.MustParse("2"),
			MemoryResource: resource.MustParse("8Gi"),
			CPU:            "2",
			Memory:         "8 GiB",
		},
		{
			Name:           "m4.xlarge",
			CPUResource:    resource.MustParse("4"),
			MemoryResource: resource.MustParse("16Gi"),
			CPU:            "2",
			Memory:         "16 GiB",
		},
	}
}

func (m *Manager) CreateWorker(ctx context.Context, mtype string) (*api.Worker, error) {
	return &api.Worker{
		ClusterName:       m.clusterName,
		MachineID:         "i-01e9c47fede75cb9a",
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       mtype,
		MachineState:      "pending",
		CreationTimestamp: time.Now(),
	}, m.err
}

func (m *Manager) GetWorker(ctx context.Context, id string) (*api.Worker, error) {
	return &api.Worker{
		ClusterName:       m.clusterName,
		MachineID:         id,
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       "m4.large",
		MachineState:      "running",
		CreationTimestamp: time.Now(),
	}, m.err
}

func (m *Manager) ListWorkers(ctx context.Context) (*api.WorkerList, error) {
	return &api.WorkerList{
		Items: []*api.Worker{
			{
				ClusterName:       m.clusterName,
				MachineID:         "i-01e9c47fededccb9a",
				MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0dededd",
				MachineType:       "m4.large",
				MachineState:      "pending",
				CreationTimestamp: time.Now(),
			},
			{
				ClusterName:       m.clusterName,
				MachineID:         "i-01e9c47fede75cb9a",
				MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
				MachineType:       "m4.large",
				MachineState:      "running",
				CreationTimestamp: time.Now(),
			},
		},
	}, m.err
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) (*api.Worker, error) {
	return &api.Worker{
		ClusterName:       m.clusterName,
		MachineID:         "i-01e9c47fede75cb9a",
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       "m4.large",
		MachineState:      "terminating",
		CreationTimestamp: time.Now(),
	}, m.err
}

func (m *Manager) ReserveWorker(ctx context.Context, w *api.Worker) (*api.Worker, error) {
	return &api.Worker{
		ClusterName:       m.clusterName,
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       "m4.large",
		MachineState:      "running",
		MachineID:         w.MachineID,
		Reserved:          w.Reserved,
		CreationTimestamp: time.Now(),
	}, m.err
}
