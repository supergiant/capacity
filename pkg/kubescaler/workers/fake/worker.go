package fake

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/provider"
)

var _ workers.WInterface = &Manager{}

type Manager struct {
	clusterName string
}

func NewManager() *Manager {
	return &Manager{
		clusterName: "fake",
	}
}

func (m *Manager) MachineTypes() []*provider.MachineType {
	return []*provider.MachineType{
		{
			Name:   "m4.large",
			CPU:    resource.MustParse("8Gi"),
			Memory: resource.MustParse("2"),
		},
		{
			Name:   "m4.xlarge",
			CPU:    resource.MustParse("16Gi"),
			Memory: resource.MustParse("4"),
		},
	}
}

func (m *Manager) CreateWorker(ctx context.Context, mtype string) (*workers.Worker, error) {
	return &workers.Worker{
		ClusterName:       m.clusterName,
		MachineID:         "i-01e9c47fede75cb9a",
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       mtype,
		MachineState:      "pending",
		CreationTimestamp: time.Now(),
	}, nil
}

func (m *Manager) ListWorkers(ctx context.Context) (*workers.WorkerList, error) {
	return &workers.WorkerList{
		Items: []*workers.Worker{
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
	}, nil
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) (*workers.Worker, error) {
	return &workers.Worker{
		ClusterName:       m.clusterName,
		MachineID:         "i-01e9c47fede75cb9a",
		MachineName:       "clusterName-worker-e289335e-9579-11e8-b97f-9cb6d0f71293",
		MachineType:       "m4.large",
		MachineState:      "terminating",
		CreationTimestamp: time.Now(),
	}, nil
}
