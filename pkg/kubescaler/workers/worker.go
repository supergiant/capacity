package workers

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/pborman/uuid"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/providers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	LabelReserved = "capacity.supergiant.io/reserved"

	LabelValueTrue  = "true"
	LabelValueFalse = "false"
)

type Config struct {
	SSHPubKey         string
	KubeVersion       string
	MasterPrivateAddr string
	KubeAPIPort       string
	KubeAPIPassword   string
	ProviderName      string
}

type Worker struct {
	ClusterName string
	MachineID   string
	MachineName string
	MachineType string
	CreatedAt   time.Time
	NodeName    string
	Reserved    bool
}

func NewWorker(node *corev1.Node) *Worker {
	return &Worker{
		MachineID: node.Spec.ProviderID,
		NodeName:  node.Name,
		Reserved:  IsReserved(node),
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

type Manager struct {
	clusterName string
	userData    string
	nodesClient v1.NodeInterface
	provider    providers.Provider
}

func NewManager(clusterName string, nodesClient v1.NodeInterface, provider providers.Provider, conf Config) (*Manager, error) {
	t, err := template.New("userData").Parse(userDataTpl)
	if err != nil {
		return nil, err
	}

	buff := &bytes.Buffer{}
	if err = t.Execute(buff, &conf); err != nil {
		return nil, err
	}

	log.Infof("worker manager: generated usedData: \n%s", buff.String())

	return &Manager{
		clusterName: clusterName,
		userData:    buff.String(),
		nodesClient: nodesClient,
		provider:    provider,
	}, nil
}

func (m *Manager) CreateWorker(ctx context.Context, mtype string) error {
	return m.provider.CreateMachine(ctx, m.clusterName, m.workerName(), mtype, m.userData, nil)
}

func (m *Manager) ListWorkers(ctx context.Context) ([]*Worker, error) {
	machines, err := m.provider.Machines(ctx)
	if err != nil {
		return nil, err
	}
	nodeProviderMap, err := m.nodesMap()
	if err != nil {
		return nil, err
	}

	workers := make([]*Worker, len(machines))
	nodeName, reserved := "", false
	for i, machine := range machines {
		if node, ok := nodeProviderMap[machine.ID]; ok {
			nodeName, reserved = node.Name, IsReserved(node)
		}

		workers[i] = &Worker{
			ClusterName: m.clusterName,
			MachineID:   machines[i].ID,
			MachineName: machines[i].Name,
			MachineType: machines[i].Type,
			CreatedAt:   machines[i].CreatedAt,
			NodeName:    nodeName,
			Reserved:    reserved,
		}
	}

	return workers, nil
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) error {
	if nodeName != "" {
		if err := m.nodesClient.Delete(nodeName, nil); err != nil {
			return err
		}
	}

	return m.provider.DeleteMachine(ctx, id)
}

func (m *Manager) nodesMap() (map[string]*corev1.Node, error) {
	nodeList, err := m.nodesClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]*corev1.Node)
	for _, node := range nodeList.Items {
		// TODO: use ProviderID instead
		if node.Spec.ExternalID != "" {
			nodeMap[node.Spec.ExternalID] = &node
		}
	}
	return nodeMap, nil
}

func (m *Manager) workerName() string {
	return fmt.Sprintf("%s-%s-%s", m.clusterName, "worker", uuid.NewUUID().String())
}
