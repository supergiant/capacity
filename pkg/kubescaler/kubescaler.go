package capacity

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kubeutil "k8s.io/autoscaler/cluster-autoscaler/utils/kubernetes"
	"k8s.io/client-go/kubernetes"

	"github.com/supergiant/capacity/pkg/kubernetes/config"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/factory"
)

const (
	// How old the oldest unschedulable pod should be before starting scale up.
	unschedulablePodTimeBuffer = 2 * time.Second
)

var (
	ErrNoAllowedMachined = errors.New("no allowed machines were provided")
)

type Kubescaler struct {
	*PersistentConfig
	*workers.Manager

	provider              provider.Provider
	kclient               kubernetes.Clientset
	listerRegistry        kubeutil.ListerRegistry
	availableMachineTypes map[string]provider.MachineType
}

func New(kubeConfig, kubescalerConfig string) (*Kubescaler, error) {
	conf, err := NewPersistentConfig(kubescalerConfig)
	if err != nil {
		return nil, err
	}

	kclient, err := config.GetKubernetesClientSet("", kubeConfig)
	if err != nil {
		return nil, err
	}

	v, err := kclient.ServerVersion()
	if err != nil {
		return nil, err
	}

	cfg := conf.GetConfig()
	vmProvider, err := factory.New(cfg.ClusterName, cfg.ProviderName, cfg.Provider)
	if err != nil {
		return nil, err
	}

	workersConf := workers.Config{
		KubeVersion:       v.String(),
		MasterPrivateAddr: cfg.MasterPrivateAddr,
		KubeAPIPort:       cfg.KubeAPIPort,
		KubeAPIPassword:   cfg.KubeAPIPassword,
		ProviderName:      cfg.ProviderName,
		SSHPubKey:         cfg.SSHPubKey,
	}
	wm, err := workers.NewManager(cfg.ClusterName, kclient.CoreV1().Nodes(), vmProvider, workersConf)
	if err != nil {
		return nil, err
	}

	return &Kubescaler{
		PersistentConfig: conf,
		Manager:          wm,
	}, nil
}

func (s *Kubescaler) RunOnce(ctx context.Context, currentTime time.Time) error {
	config := s.GetConfig()
	// TODO: turn on after e2e testing
	//if config.Stopped {
	if true {
		return nil
	}

	rss, err := s.getResources(ctx)
	if err != nil {
		return err
	}

	// remove machines that are provisioning for a long time
	removed, err := s.removeFailedMachines(ctx, rss, currentTime)
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

	if config.NodesCountMax >= len(rss.readyNodes) {
		// try to scale up the cluster. In case of success no need to scale down
		if err = s.scaleUp(ctx, rss.unschedulablePods, rss.readyNodes, s.machineTypes(config.MachineTypes), currentTime); err != nil {
			return err
		}
	}

	if config.NodesCountMin < len(rss.readyNodes) {
		if err = s.scaleDown(ctx, rss.scheduledPods, rss.readyNodes); err != nil {
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

func (s *Kubescaler) getResources(ctx context.Context) (*resources, error) {
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
	rss.machines, err = s.provider.Machines(ctx)
	if err != nil {
		return nil, err
	}

	return &rss, nil
}

func (s *Kubescaler) removeFailedMachines(ctx context.Context, rss *resources, currentTime time.Time) (bool, error) {
	var fixed bool
	if len(rss.machines) == len(rss.readyNodes) {
		return fixed, nil
	}

	for _, m := range rss.machines {
		if m.CreatedAt.Add(s.GetConfig().MaxMachineProvisionTime).Before(currentTime) {
			if _, err := s.provider.DeleteMachine(ctx, m.ID); err != nil {
				return fixed, err
			}
			fixed = true
		}
	}

	return fixed, nil
}

func (s *Kubescaler) machineTypes(types []string) []*provider.MachineType {
	out := make([]*provider.MachineType, 0, len(types))
	for _, t := range types {
		if mt, ok := s.availableMachineTypes[t]; ok {
			out = append(out, &mt)
		}
	}
	return out
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
