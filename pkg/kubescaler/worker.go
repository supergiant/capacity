package capacity

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/supergiant/capacity/pkg/provider"
)

const (
	LabelReserved = "capacity.supergiant.io/reserved"

	LabelValueTrue  = "true"
	LabelValueFalse = "false"
)

type Worker struct {
	MachineName string
	NodeName    string
	Reserved    bool
}

func NewWorker(node *corev1.Node) *Worker {
	return &Worker{
		MachineName: node.Spec.ProviderID,
		NodeName:    node.Name,
		Reserved:    IsReserved(node),
	}
}

func IsReserved(node *corev1.Node) bool {
	if v, ok := node.Labels[LabelReserved]; ok {
		if strings.ToLower(v) == LabelValueTrue {
			return true
		}
	}
	return false
}

type WorkerManager struct {
	nodesClient v1.NodeInterface
	provider    provider.Provider
}

func (r *WorkerManager) Create(machineType string) error {
	_, err := r.provider.CreateMachine(machineType)
	return err
}

func (r *WorkerManager) Get(name string) (*Worker, error) {
	node, err := r.nodesClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return NewWorker(node), nil
}

func (r *WorkerManager) Delete(name string, force bool) error {
	w, err := r.Get(name)
	if err != nil {
		return err
	}

	if !force {
		if err := r.nodesClient.Delete(w.NodeName, nil); err != nil {
			return err
		}
	}

	return r.provider.DeleteMachine(w.MachineName)
}
