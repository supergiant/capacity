package workers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/supergiant/capacity/pkg/provider"
)

const (
	LabelReserved = "capacity.supergiant.io/reserved"

	ValTrue  = "true"
	ValFalse = "false"

	ClusterRole = "worker"

	MinWorkerLifespan = time.Minute * 20
)

var _ WInterface = &Manager{}

type WInterface interface {
	MachineTypes() []*provider.MachineType
	CreateWorker(ctx context.Context, mtype string) (*Worker, error)
	ListWorkers(ctx context.Context) (*WorkerList, error)
	DeleteWorker(ctx context.Context, nodeName, id string) (*Worker, error)
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

// Worker is an abstraction used by kubescaler to manage cluster capacity.
// It contains data from a (virtual) machine and a kubernetes node running on it.
type Worker struct {
	// ClusterName is a kubernetes cluster name.
	ClusterName string `json:"clusterName"`
	// MachineID is a unique id of the provider's virtual machine.
	// required: true
	MachineID string `json:"machineID"`
	// MachineName is a human-readable name of virtual machine.
	MachineName string `json:"machineName"`
	// MachineType is type of virtual machine (eg. 't2.micro' for AWS).
	MachineType string `json:"machineType"`
	// MachineState represent a virtual machine state.
	MachineState string `json:"machineState"`
	// CreationTimestamp is a timestamp representing a time when this machine was created.
	CreationTimestamp time.Time `json:"creationTimestamp"`
	// Reserved is a parameter that is used to prevent downscaling of the worker.
	Reserved bool `json:"reserved"`
	// NodeName represents a name of the kubernetes node that runs on top of that machine.
	NodeName string `json:"nodeName"`
	// NodeLabels represents a labels of the kubernetes node that runs on top of that machine.
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`
}

type WorkerList struct {
	Items []*Worker `json:"items"`
}

func NewWorker(node *corev1.Node) *Worker {
	return &Worker{
		MachineID: node.Spec.ExternalID,
		NodeName:  node.Name,
		Reserved:  IsReserved(node),
	}
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

	var templ []byte
	//if userdata was provided
	if conf.UserDataFile != "" {
		//read userdata file
		templ, err = ioutil.ReadFile(conf.UserDataFile)
		if err != nil {
			return nil, err
		}
	} else { //If no userdata was provided
		//use the default userdata
		templ = []byte(userDataTpl)
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

func (m *Manager) CreateWorker(ctx context.Context, mtype string) (*Worker, error) {
	machine, err := m.provider.CreateMachine(ctx, m.workerName(), mtype, ClusterRole, m.userData, nil)
	if err != nil {
		return nil, err
	}
	return m.workerFrom(machine, corev1.Node{}), nil
}

func (m *Manager) MachineTypes() []*provider.MachineType {
	return m.machineTypes
}

func (m *Manager) ListWorkers(ctx context.Context) (*WorkerList, error) {
	machines, err := m.provider.Machines(ctx)
	if err != nil {
		return nil, err
	}
	nodesMap, err := m.nodesMap()
	if err != nil {
		return nil, err
	}

	workers := make([]*Worker, len(machines))
	for i := range machines {
		workers[i] = m.workerFrom(machines[i], nodesMap[machines[i].ID])
	}

	return &WorkerList{
		Items: workers,
	}, nil
}

func (m *Manager) DeleteWorker(ctx context.Context, nodeName, id string) (*Worker, error) {
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

func (m *Manager) nodesMap() (map[string]corev1.Node, error) {
	nodeList, err := m.nodesClient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]corev1.Node)
	for _, node := range nodeList.Items {
		machineID, err := m.provider.GetMachineID(node.Spec.ProviderID)
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

func (m *Manager) workerFrom(machine *provider.Machine, node corev1.Node) *Worker {
	return &Worker{
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
