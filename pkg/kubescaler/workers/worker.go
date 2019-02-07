package workers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/provider"
)

const (
	LabelReserved = "capacity.supergiant.io/reserved"
	ValTrue       = "true"

	ClusterRole = "worker"

	MinWorkerLifespan = time.Minute * 20
)

var (
	ErrNotFound = errors.New("not found")
)

type WInterface interface {
	MachineTypes() []*provider.MachineType
	CreateWorker(ctx context.Context, mtype string) (*api.Worker, error)
	GetWorker(ctx context.Context, id string) (*api.Worker, error)
	ListWorkers(ctx context.Context) (*api.WorkerList, error)
	DeleteWorker(ctx context.Context, nodeName, id string) (*api.Worker, error)
	ReserveWorker(ctx context.Context, worker *api.Worker) (*api.Worker, error)
}

type Config struct {
	SSHPubKey         string
	KubeVersion       string
	MasterPrivateAddr string
	KubeAPIPort       string
	KubeAPIPassword   string
	ProviderName      string
	UserDataFile      string
}

func IsReserved(node *corev1.Node) bool {
	if v, ok := node.Labels[LabelReserved]; ok {
		if strings.ToLower(v) == ValTrue {
			return true
		}
	}
	return false
}

type Manager struct {
	clusterName  string
	userData     string
	nodesClient  v1.NodeInterface
	provider     provider.Provider
	machineTypes []*provider.MachineType
}

func NewManager(clusterName string, nodesClient v1.NodeInterface, provider provider.Provider, conf Config) (*Manager, error) {
	mtypes, err := provider.MachineTypes(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "get machine types")
	}

	templ := []byte(userDataTpl)
	//if userdata was provided
	if conf.UserDataFile != "" {
		//read userdata file
		templ, err = ioutil.ReadFile(conf.UserDataFile)
		if err != nil {
			return nil, err
		}
	}

	t, err := template.New("userData").Parse(string(templ))
	if err != nil {
		return nil, err
	}

	buff := &bytes.Buffer{}
	if err = t.Execute(buff, &conf); err != nil {
		return nil, err
	}

	//log.Debugf("worker manager: generated usedData: \n%s", buff.String())

	return &Manager{
		clusterName:  clusterName,
		userData:     buff.String(),
		nodesClient:  nodesClient,
		provider:     provider,
		machineTypes: mtypes,
	}, nil
}

func (m *Manager) CreateWorker(ctx context.Context, mtype string) (*api.Worker, error) {
	machine, err := m.provider.CreateMachine(ctx, m.workerName(), mtype, ClusterRole, m.userData, nil)
	if err != nil {
		return nil, err
	}
	return m.workerFrom(machine, corev1.Node{}), nil
}

func (m *Manager) MachineTypes() []*provider.MachineType {
	return m.machineTypes
}

func (m *Manager) GetWorker(ctx context.Context, id string) (*api.Worker, error) {
	machine, err := m.provider.GetMachine(ctx, id)
	if err != nil {
		return nil, err
	}

	node, err := m.getNodeByMachine(id)
	if err != nil {
		return nil, err
	}

	return m.workerFrom(machine, node), nil
}

func (m *Manager) ListWorkers(ctx context.Context) (*api.WorkerList, error) {
	machines, err := m.provider.Machines(ctx)
	if err != nil {
		return nil, err
	}
	nodesMap, err := m.nodesMap()
	if err != nil {
		return nil, err
	}

	workers := make([]*api.Worker, len(machines))
	for i := range machines {
		workers[i] = m.workerFrom(machines[i], nodesMap[machines[i].ID])
	}

	return &api.WorkerList{
		Items: workers,
	}, nil
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) (*api.Worker, error) {
	if nodeName != "" {
		if err := m.nodesClient.Delete(nodeName, nil); err != nil {
			return nil, err
		}
	}

	machine, err := m.provider.DeleteMachine(ctx, id)
	if err != nil {
		return nil, err
	}

	return m.workerFrom(machine, corev1.Node{}), nil
}

func (m *Manager) ReserveWorker(ctx context.Context, want *api.Worker) (*api.Worker, error) {
	if want == nil {
		return nil, ErrNotFound
	}

	current, err := m.GetWorker(ctx, want.MachineID)
	if err != nil {
		return nil, err
	}

	if current.Reserved == want.Reserved {
		return current, nil
	}

	return m.setReserved(current, want.Reserved)
}

func (m *Manager) getNodeByMachine(id string) (corev1.Node, error) {
	instanceNodesMap, err := m.nodesMap()
	if err != nil {
		return corev1.Node{}, err
	}
	return instanceNodesMap[id], nil
}

func (m *Manager) nodesMap() (map[string]corev1.Node, error) {
	nodeList, err := m.nodesClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]corev1.Node)
	for _, node := range nodeList.Items {
		machineID, err := m.provider.ParseMachineID(node.Spec.ProviderID)
		if err != nil {
			return nil, errors.Wrap(err, "parse node.Spec.ProviderID")
		}
		nodeMap[machineID] = node
	}
	return nodeMap, nil
}

func (m *Manager) workerName() string {
	return fmt.Sprintf("%s-%s-%s", m.clusterName, "worker", uuid.NewUUID().String())
}

func (m *Manager) workerFrom(machine *provider.Machine, node corev1.Node) *api.Worker {
	return &api.Worker{
		ClusterName:       m.clusterName,
		MachineID:         machine.ID,
		MachineName:       machine.Name,
		MachineType:       machine.Type,
		MachineState:      machine.State,
		CreationTimestamp: machine.CreationTimestamp,
		Reserved:          IsReserved(&node),
		NodeName:          node.Name,
		NodeLabels:        node.Labels,
	}
}

func (m *Manager) setReserved(w *api.Worker, reserved bool) (*api.Worker, error) {
	node, err := m.patchNodeLabel(w.NodeName, LabelReserved, strconv.FormatBool(reserved))
	if err != nil {
		return nil, err
	}
	// TODO: add a method for updating
	w.NodeLabels = node.Labels
	w.Reserved = reserved
	return w, nil
}

func (m *Manager) patchNodeLabel(nodeName, key, val string) (*corev1.Node, error) {
	return m.nodesClient.Patch(nodeName, types.MergePatchType,
		[]byte(fmt.Sprintf(`{"metadata":{"labels":{%q:%q}}}`, key, val)))
}
