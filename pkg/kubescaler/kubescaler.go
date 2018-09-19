package capacity

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeutil "k8s.io/autoscaler/cluster-autoscaler/utils/kubernetes"
	"k8s.io/client-go/kubernetes"

	"github.com/supergiant/capacity/pkg/kubernetes/config"
	"github.com/supergiant/capacity/pkg/kubernetes/filters"
	"github.com/supergiant/capacity/pkg/kubernetes/listers"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/kubescaler/workers/fake"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/factory"
)

const (
	// How old the oldest unschedulable pod should be before starting scale up.
	unschedulablePodTimeBuffer = 2 * time.Second
)

var (
	ErrNoAllowedMachines = errors.New("no allowed machines were provided")

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
	listerRegistry listers.Registry
}

func New(kubeConfig, kubescalerConfig, userDataFile string) (*Kubescaler, error) {
	conf, err := NewPersistentConfig(kubescalerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "build config")
	}
	cfg := conf.GetConfig()

	// use a fake kubescaler for testing
	if conf.GetConfig().ProviderName == "fake" {
		return &Kubescaler{
			PersistentConfig: conf,
			WInterface:       fake.NewManager(),
		}, nil
	}

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
		UserDataFile:      userDataFile,
	}
	wm, err := workers.NewManager(cfg.ClusterName, kclient.CoreV1().Nodes(), vmProvider, workersConf)
	if err != nil {
		return nil, err
	}

	return &Kubescaler{
		PersistentConfig: conf,
		WInterface:       wm,
		listerRegistry:   listers.NewRegistryWithDefaultListers(kclient, nil),
	}, nil
}

func (s *Kubescaler) Run(stop <-chan struct{}) {
	log.Info("starting kubescaler...")
	pauseLockCheck := s.GetConfig()
	//checking to see if pauselock is engaged.
	//We do this check here so the Warn will not eat up logs in the RunOnce func.
	if pauseLockCheck.PauseLock == true {
		log.Warn("Pause Lock engaged. Automatic Capacity will not occur.")
	} else {
		log.Info("Automatic Capacity will occur unless paused.")
	}

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

	//Paused defaults to false if omitted.
	paused := config.Paused != nil && *(config.Paused)
	pauseLocked := config.PauseLock
	if paused && !pauseLocked {
		log.Info("Service is paused.")
	}

	if paused || pauseLocked {
		//dont do auto scaling
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
		if newNodes := getNewNodes(rss.allNodes, currentTime, config.NewNodeTimeBuffer); len(newNodes) != 0 {
			log.Debugf("kubescaler: scale up: newNodes=%v, skipping", newNodes)
			return nil
		}

		nodePods := nodePodsMap(rss.scheduledPods)
		log.Debugf("kubescaler: scale up: nodepods %v, ready nodes %v", nodePods, nodeNames(rss.readyNodes))
		if len(rss.readyNodes) < len(nodePods) {
			// have some scheduled pods (pending|ready) that have been already scheduled on nodes (not ready yet).
			return nil
		}

		if emptyNodes := getEmptyNodes(rss.readyNodes, rss.allPods); len(emptyNodes) > 0 {
			log.Debugf("kubescaler: scale up: there are %v ready empty nodes in the cluster", nodeNames(emptyNodes))
			return nil
		}

		// TODO: use workers instead of nodes (workerList may contain 'terminating' machines)
		if config.WorkersCountMax > 0 && config.WorkersCountMax > len(rss.readyNodes) {
			var scaled bool
			// try to scale up the cluster. In case of success no need to scale down
			scaled, err = s.scaleUp(rss.unscheduledPods, allowedMachineTypes, currentTime)
			if err != nil {
				return errors.Wrap(err, "scale up")
			}
			if scaled {
				return nil
			}
		} else {
			log.Debugf("kubescaler: scaleup: workersCountMax(%d) >= number of ready nodes(%d), skipping..",
				config.WorkersCountMax, len(rss.readyNodes))
		}
	}

	// TODO: workerList may contain 'terminating' machines.
	if config.WorkersCountMin > 0 && config.WorkersCountMin < len(rss.readyNodes) {
		if err = s.scaleDown(rss.scheduledPods, rss.workerList, config.IgnoredNodeLabels, currentTime); err != nil {
			return errors.Wrap(err, "scale down")
		}
	} else {
		log.Debugf("kubescaler: scaledown: workersCountMin(%d) < number of ready nodes(%d), skipping..",
			config.WorkersCountMin, len(rss.readyNodes))
	}

	return nil
}

type resources struct {
	allNodes        []*corev1.Node
	readyNodes      []*corev1.Node
	allPods         []*corev1.Pod
	scheduledPods   []*corev1.Pod
	unscheduledPods []*corev1.Pod
	workerList      *workers.WorkerList
}

func (s *Kubescaler) getResources() (*resources, error) {
	allNodes, err := s.listerRegistry.AllNodeLister().List()
	if err != nil {
		return nil, err
	}

	allPods, err := s.listerRegistry.AllPodLister().List()
	if err != nil {
		return nil, err
	}

	workers, err := s.ListWorkers(context.Background())
	if err != nil {
		return nil, err
	}

	return &resources{
		allNodes:        allNodes,
		readyNodes:      filters.GetReadyNodes(allNodes),
		allPods:         allPods,
		scheduledPods:   filters.GetScheduledPods(allPods),
		unscheduledPods: filters.GetUnschedulablePods(allPods),
		workerList:      workers,
	}, nil
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

func nodeNames(nodes []*corev1.Node) []string {
	list := make([]string, len(nodes))
	for i := range nodes {
		list[i] = nodes[i].Name
	}
	return list
}

const (
	nodeLabelRole   = "kubernetes.io/role"
	nodeLabelMaster = "master"
)

func filterOutMasters(nodes []*corev1.Node, pods []*corev1.Pod) []*corev1.Node {
	masters := make(map[string]bool)
	for _, pod := range pods {
		if pod.Namespace == metav1.NamespaceSystem && pod.Labels[nodeLabelRole] == nodeLabelMaster {
			masters[pod.Spec.NodeName] = true
		}
	}

	// if masters aren't on the list of nodes, capacity will be increased on overflowing append
	others := make([]*corev1.Node, 0, len(nodes)-len(masters))
	for _, node := range nodes {
		if !masters[node.Name] {
			others = append(others, node)
		}
	}

	return others
}

// getEmptyNodes filter out nodes that have at least one pod scheduled on it (node.name == pod.spec.nodeName).
func getEmptyNodes(nodes []*corev1.Node, pods []*corev1.Pod) []*corev1.Node {
	nodePods := nodePodsMap(pods)
	emptyNodes := make([]*corev1.Node, 0)
	for _, node := range nodes {
		if len(nodePods[node.Name]) == 0 {
			emptyNodes = append(emptyNodes, node)
		}
	}
	return emptyNodes
}

func getNewNodes(nodes []*corev1.Node, currentTime time.Time, newNodeTimeBuffer int) []*corev1.Node {
	newNodes := make([]*corev1.Node, 0)
	for _, node := range nodes {
		if isNewNode(node, currentTime, newNodeTimeBuffer) {
			newNodes = append(newNodes, node)
		}
	}
	return newNodes
}

func isNewNode(node *corev1.Node, currentTime time.Time, newNodeTimeBuffer int) bool {
	return node.CreationTimestamp.Add(time.Duration(newNodeTimeBuffer) * time.Second).After(currentTime)
}
