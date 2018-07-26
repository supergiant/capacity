package capacity

import (
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kubeutil "k8s.io/autoscaler/cluster-autoscaler/utils/kubernetes"
	"k8s.io/client-go/kubernetes"

	"github.com/supergiant/capacity/pkg/provider"
)

const (
	// How old the oldest unschedulable pod should be before starting scale up.
	unschedulablePodTimeBuffer = 2 * time.Second
)

var (
	ErrNoAllowedMachined = errors.New("no allowed machines were provided")
)

type Config struct {
	NodesCountMin int
	NodesCountMax int

	MaxMachineProvisionTime time.Duration

	AllowedMachines []provider.MachineType
}

type Kubescaler struct {
	config         Config
	provider       provider.Provider
	kclient        kubernetes.Clientset
	listerRegistry kubeutil.ListerRegistry
	workerManager  *WorkerManager
}

func New() (*Kubescaler, error) {
	return nil, nil
}

func (s *Kubescaler) GetWorker(name string) (*Worker, error) {
	return s.workerManager.Get(name)
}

func (s *Kubescaler) DeleteWorker(name string, force bool) error {
	return s.workerManager.Delete(name, force)
}

func (s *Kubescaler) RunOnce(currentTime time.Time) error {
	rss, err := s.getResources()
	if err != nil {
		return err
	}

	// remove machines that are provisioning for a long time
	removed, err := s.removeFailedMachines(rss, currentTime)
	if err != nil {
		return err
	}
	if removed {
		return nil
	}

	if len(provisioningMachines(rss.readyNodes, rss.machines)) != 0 {
		// some machines are provisioning now, wait for them to be ready
		// skip scale up/down until all of them are ready
		return nil
	}

	if s.config.NodesCountMax >= len(rss.readyNodes) {
		// try to scale up the cluster. In case of success no need to scale down
		if err = s.scaleUp(rss.unschedulablePods, rss.readyNodes, currentTime); err != nil {
			return err
		}
	}

	if s.config.NodesCountMin < len(rss.readyNodes) {
		if err = s.scaleDown(rss.scheduledPods, rss.readyNodes); err != nil {
			return err
		}
	}

	return nil
}

type resources struct {
	scheduledPods     []*corev1.Pod
	unschedulablePods []*corev1.Pod
	nodes             []*corev1.Node
	readyNodes        []*corev1.Node
	machines          []*provider.Machine
}

func (s *Kubescaler) getResources() (*resources, error) {
	var rss resources
	var err error

	rss.scheduledPods, err = s.listerRegistry.ScheduledPodLister().List()
	if err != nil {
		return nil, err
	}
	rss.unschedulablePods, err = s.listerRegistry.UnschedulablePodLister().List()
	if err != nil {
		return nil, err
	}
	rss.nodes, err = s.listerRegistry.AllNodeLister().List()
	if err != nil {
		return nil, err
	}
	rss.readyNodes, err = s.listerRegistry.ReadyNodeLister().List()
	if err != nil {
		return nil, err
	}
	rss.machines, err = s.provider.Machines()
	if err != nil {
		return nil, err
	}

	return &rss, nil
}

func (s *Kubescaler) removeFailedMachines(rss *resources, currentTime time.Time) (bool, error) {
	var fixed bool
	if len(rss.machines) == len(rss.readyNodes) {
		return fixed, nil
	}

	for _, m := range rss.machines {
		if m.CreatedAt.Add(s.config.MaxMachineProvisionTime).Before(currentTime) {
			if err := s.provider.DeleteMachine(m.ID); err != nil {
				return fixed, err
			}
			fixed = true
		}
	}

	return fixed, nil
}

func provisioningMachines(readyNodes []*corev1.Node, machines []*provider.Machine) []*provider.Machine {
	registered := sets.NewString()
	for _, node := range readyNodes {
		registered.Insert(node.Spec.ProviderID)
	}
	unregistered := make([]*provider.Machine, 0)
	for _, machine := range machines {
		if !registered.Has(machine.ID) {
			unregistered = append(unregistered, machine)
		}
	}
	return unregistered
}
