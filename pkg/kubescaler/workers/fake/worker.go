package fake

import (
	"context"
	"time"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
)

type Manager struct {
	clusterName string
}

func NewManager() *Manager {
	return &Manager{
		clusterName: "fake",
	}
}

func (m *Manager) CreateWorker(ctx context.Context, mtype string) (*workers.Worker, error) {
	return &workers.Worker{
		ClusterName:  m.clusterName,
		MachineID:    "i-01e9c47fede75cb9a",
		MachineName:  "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:  mtype,
		MachineState: "pending",
		CreatedAt:    time.Now(),
	}, nil
}

func (m *Manager) ListWorkers(ctx context.Context) ([]*workers.Worker, error) {
	return []*workers.Worker{
		{
			ClusterName:  m.clusterName,
			MachineID:    "i-01e9c47fededccb9a",
			MachineName:  "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0dededd",
			MachineType:  "m4.large",
			MachineState: "pending",
			CreatedAt:    time.Now(),
		},
		{
			ClusterName:  m.clusterName,
			MachineID:    "i-01e9c47fede75cb9a",
			MachineName:  "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
			MachineType:  "m4.large",
			MachineState: "running",
			CreatedAt:    time.Now(),
		},
	}, nil
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) (*workers.Worker, error) {
	return &workers.Worker{
		ClusterName:  m.clusterName,
		MachineID:    "i-01e9c47fede75cb9a",
		MachineName:  "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:  "m4.large",
		MachineState: "terminating",
		CreatedAt:    time.Now(),
	}, nil
}
