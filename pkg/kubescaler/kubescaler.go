package capacity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeutil "k8s.io/autoscaler/cluster-autoscaler/utils/kubernetes"
	"k8s.io/client-go/kubernetes"

	"github.com/supergiant/capacity/pkg/kubernetes/config"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/factory"
)

const (
	// How old the oldest unschedulable pod should be before starting scale up.
	unschedulablePodTimeBuffer = 2 * time.Second

	newNodeTimeBuffer = 3 * time.Minute
)

var (
	ErrNoAllowedMachined = errors.New("no allowed machines were provided")

	DefaultScanInterval            = time.Second * 20
	DefaultMaxMachineProvisionTime = time.Minute * 10
)

type ListerRegistry interface {
	ReadyNodeLister() kubeutil.NodeLister
	ScheduledPodLister() kubeutil.PodLister
	UnschedulablePodLister() kubeutil.PodLister
}

type Kubescaler struct {
	*PersistentConfig
	workers.WInterface

	kclient        kubernetes.Clientset
	listerRegistry ListerRegistry
}

func New(kubeConfig, kubescalerConfig string) (*Kubescaler, error) {
	conf, err := NewPersistentConfig(kubescalerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "build config")
	}
	cfg := conf.GetConfig()

	var ks *Kubescaler
	if conf.GetConfig().ProviderName == "fake" {
		ks = &Kubescaler{
			PersistentConfig: conf,
			WInterface:       fake.NewManager(),
		}
	} else {
		kclient, err := config.GetKubernetesClientSetBasicAuth(cfg.KubeAPIHost, cfg.KubeAPIPort, cfg.KubeAPIUser, cfg.KubeAPIPassword)
		if err != nil {
			return nil, err
		}

		v, err := kclient.ServerVersion()
		if err != nil {
			return nil, err
		}

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

		ks = &Kubescaler{
			PersistentConfig: conf,
			WInterface:       wm,
			// TODO: implement a cached lister registry
			listerRegistry: kubeutil.NewListerRegistry(
				nil,
				kubeutil.NewReadyNodeLister(kclient, nil),
				kubeutil.NewScheduledPodLister(kclient, nil),
				kubeutil.NewUnschedulablePodLister(kclient, nil),
				nil,
				nil,
			),
		}
	}

	return ks, nil
}

func (s *Kubescaler) Run(stop <-chan struct{}) {
	log.Info("starting kubescaler...")

	go func() {
		for {
			select {
			case <-time.After(DefaultScanInterval):
				{
					if err := s.RunOnce(time.Now()); err != nil {
						log.Errorf("kubescaler: %v", err)
					}
				}
			case <-stop:
				return
			}
		}
	}()
}

func (s *Kubescaler) RunOnce(currentTime time.Time) error {
	config := s.GetConfig()
	// TODO: turn on after e2e testing
	if config.Paused != nil && *config.Paused {
		//if true {
		return nil
	}

	allowedMachineTypes := s.machineTypes(config.MachineTypes)
	if len(allowedMachineTypes) == 0 {
		log.Error("kubescaler: node available machine types we found; please, check the configuration")
		return nil
	}

	rss, err := s.getResources()
	if err != nil {
		return err
	}

	log.Debugf("kubescaler: rss: nodes=%v unscheduledPods=%v", workerNodeNames(rss.workerList.Items), podNames(rss.unscheduledPods))

	failed, provisioning := s.checkWorkers(rss.workerList, currentTime)
	if len(failed) > 0 {
		// remove machines that are provisioning for a long time
		log.Debugf("kubescaler: removing %s failed machines", failed)
		return s.removeFailedMachines(failed)
	}
	if len(provisioning) > 0 {
		// some machines are provisioning now, wait for them to be ready
		// skip scale up/down until all of them are ready
		log.Debugf("kubescaler: %v machines are provisioning now", provisioning)
		return nil
	}

	if len(rss.unscheduledPods) > 0 {
		// TODO: worker manager uses kube client, readyNodes - nodeLister...
		if len(workerNodeNames(rss.workerList.Items)) != len(rss.readyNodes) {
			log.Debugf("kubescaler: scale up: workerNodes(%d) != readyNodes(%d)",
				len(workerNodeNames(rss.workerList.Items)), len(rss.readyNodes))
			return nil
		}
		// TODO: node has created, but controller doesn't assign a pod to it yet
		log.Debugf("kubescaler: scale up: nodepods %#v, ready nodes %v", nodePodMap(rss.scheduledPods), nodeNames(rss.readyNodes))
		if len(rss.readyNodes) != len(nodePodMap(rss.scheduledPods)) {
			log.Debugf("kubescaler: scale up: there are empty nodes in cluster")
			return nil
		}

		// TODO: use workers instead of nodes (workerList may contain 'terminating' machines)
		if config.WorkersCountMax >= len(rss.readyNodes) {
			// try to scale up the cluster. In case of success no need to scale down
			scaled, err := s.scaleUp(rss.unscheduledPods, allowedMachineTypes, currentTime)
			if err != nil {
				return errors.Wrap(err, "scale up")
			}
			if scaled {
				return nil
			}
		} else {
			log.Debugf("kubescaler: scaleup: workersCountMax(%d) >= number of workers(%d), skipping..",
				config.WorkersCountMax, len(rss.readyNodes))
		}
	}

	// TODO: workerList may contain 'terminating' machines.
	if config.WorkersCountMin < len(rss.readyNodes) {
		if err = s.scaleDown(rss.scheduledPods, rss.workerList, currentTime); err != nil {
			return errors.Wrap(err, "scale down")
		}
	} else {
		log.Debugf("kubescaler: scaledown: workersCountMin(%d) < number of workers(%d), skipping..",
			config.WorkersCountMin, len(rss.readyNodes))
	}

	return nil
}

type resources struct {
	readyNodes      []*corev1.Node
	scheduledPods   []*corev1.Pod
	unscheduledPods []*corev1.Pod
	workerList      *workers.WorkerList
}

func (s *Kubescaler) getResources() (*resources, error) {
	var rss resources
	var err error

	rss.readyNodes, err = s.listerRegistry.ReadyNodeLister().List()
	if err != nil {
		return nil, err
	}
	rss.scheduledPods, err = s.listerRegistry.ScheduledPodLister().List()
	if err != nil {
		return nil, err
	}
	rss.unscheduledPods, err = s.listerRegistry.UnschedulablePodLister().List()
	if err != nil {
		return nil, err
	}
	rss.workerList, err = s.ListWorkers(context.Background())
	if err != nil {
		return nil, err
	}

	return &rss, nil
}

func (s *Kubescaler) checkWorkers(workerList *workers.WorkerList, currentTime time.Time) ([]string, []string) {
	//failedMachines:
	//	- state == 'running'
	//	- running >= maxProvisionTime
	//	- have no registered node, skip master
	failed := make([]string, 0)

	//	provisioning machines:
	//	- state == 'pending' || 'running', https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html
	//	- running <= maxProvisionTime
	//	- have no registered node, skip master
	provisioning := make([]string, 0)

	for _, worker := range workerList.Items {
		ignored := !(worker.MachineState == "pending" || worker.MachineState == "running") ||
			worker.NodeName != "" ||
			isMaster(worker)
		if ignored {
			continue
		}

		if worker.CreationTimestamp.Add(DefaultMaxMachineProvisionTime).Before(currentTime) {
			failed = append(failed, worker.MachineID)
		} else {
			provisioning = append(provisioning, worker.MachineID)
		}
	}

	return failed, provisioning
}

func (s *Kubescaler) removeFailedMachines(ids []string) error {
	for _, id := range ids {
		if _, err := s.DeleteWorker(context.Background(), "", id); err != nil {
			return err
		}

	}
	return nil
}

func (s *Kubescaler) machineTypes(permitted []string) []*provider.MachineType {
	out := make([]*provider.MachineType, 0, len(permitted))
	for _, name := range permitted {
		if mt := findMachine(name, s.WInterface.MachineTypes()); mt != nil {
			out = append(out, mt)
		}
	}

	return out

}

func findMachine(name string, machineTypes []*provider.MachineType) *provider.MachineType {
	for i := range machineTypes {
		if name == machineTypes[i].Name {
			return machineTypes[i]
		}
	}
	return nil
}

func isMaster(w *workers.Worker) bool {
	// TODO: use role tags for it in SG2.0
	return strings.Contains(strings.ToLower(w.MachineName), "master")
}

// don't work if server time isn't synced
func getNewNodes(nodes []*corev1.Node, currentTime time.Time) []string {
	newNodes := make([]string, 0)
	for _, node := range nodes {
		if node.CreationTimestamp.Add(newNodeTimeBuffer).After(currentTime) {
			newNodes = append(newNodes, fmt.Sprintf("%s(%s)", node.Name, currentTime.Sub(node.CreationTimestamp.Time)))
		}
	}
	return newNodes
}

func nodeNames(nodes []*corev1.Node) []string {
	list := make([]string, len(nodes))
	for i := range nodes {
		list[i] = nodes[i].Name
	}
	return list
}
