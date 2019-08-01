package kubescaler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeversion "k8s.io/apimachinery/pkg/version"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/kubernetes/config"
	"github.com/supergiant/capacity/pkg/kubernetes/filters"
	"github.com/supergiant/capacity/pkg/kubernetes/listers"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/persistentfile"
	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/factory"
)

const (
	DefaultConfigFilepath = "/etc/kubescaler.conf"

	// How old the oldest unschedulable pod should be before starting scale up.
	unschedulablePodTimeBuffer = 2 * time.Second

	// defaultNodeTimeBuffer is a time in seconds to wait for node is in Ready state.
	defaultNodeTimeBuffer = 60
)

var (
	ErrNoAllowedMachines = errors.New("no allowed machines were provided")
	ErrNotConfigured     = errors.New("worker manager is not configured properly")

	DefaultScanInterval            = time.Second * 20
	DefaultMaxMachineProvisionTime = time.Minute * 10
)

type ListerRegistry interface {
	ReadyNodeLister() listers.NodeLister
	ScheduledPodLister() listers.PodLister
	UnschedulablePodLister() listers.PodLister
}

type Options struct {
	ConfigFile         string
	ConfigMapName      string
	ConfigMapNamespace string
	Kubeconfig         string
}

type Kubescaler struct {
	stopCh         chan struct{}
	kclient        corev1client.CoreV1Interface
	listerRegistry listers.Registry

	configManager *ConfigManager

	workerMutex   sync.RWMutex
	isReady       bool
	workerManager workers.WInterface
}

func New(opts Options) (*Kubescaler, error) {
	// TODO: use corev1 client
	kclient, err := config.GetCoreV1Client("", opts.Kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "build kubernetes client")
	}

	f, err := getConfigFile(opts, kclient)
	if err != nil {
		return nil, err
	}
	log.Infof("kubescaler: get config from: %s", f.Info())

	conf, err := NewConfigManager(f)
	if err != nil {
		return nil, errors.Wrap(err, "setup persistent config")
	}

	kubeScaler := &Kubescaler{
		kclient:        kclient,
		configManager:  conf,
		stopCh:         make(chan struct{}),
		listerRegistry: listers.NewRegistryWithDefaultListers(kclient.RESTClient(), nil),
	}

	// We skip this error because on this stage capacity service may not be
	// configured
	if err := kubeScaler.buildWorkerManager(); err != nil {
		log.Infof("kubescaler was not configured yet, to configure " +
			"make POST request to /api/v1/config Handler with valid configHandler object")
	}

	return kubeScaler, nil
}

func (s *Kubescaler) Run() error {
	pauseLockCheck := s.configManager.GetConfig()
	//checking to see if pauselock is engaged.
	//We do this check here so the Warn will not eat up logs in the RunOnce func.
	if pauseLockCheck.PauseLock == true {
		log.Warn("Pause Lock engaged. Automatic Capacity will not occur.")
	} else {
		log.Info("Automatic Capacity will occur unless paused.")
	}

	func() {
		for {
			select {
			case <-time.After(DefaultScanInterval):
				{
					if !s.IsReady() {
						continue
					}
					if err := s.RunOnce(time.Now()); err != nil {
						log.Errorf("kubescaler: %v", err)
					}
				}
			case <-s.stopCh:
				return
			}
		}
	}()

	return nil
}

func (s *Kubescaler) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		// stop chan is synchronous, waiting for receiver
		s.stopCh <- struct{}{}
		done <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		}
	}
}

func (s *Kubescaler) RunOnce(currentTime time.Time) error {
	cfg := s.configManager.GetConfig()

	//Paused defaults to false if omitted.
	paused := cfg.Paused != nil && *(cfg.Paused)
	pauseLocked := cfg.PauseLock
	if paused && !pauseLocked {
		log.Info("Service is paused.")
	}

	if paused || pauseLocked {
		//dont do auto scaling
		return nil
	}

	allowedMachineTypes := s.machineTypes(cfg.MachineTypes)
	if len(allowedMachineTypes) == 0 {
		log.Error("kubescaler: node available machine types we found; please, check the configuration")
		return nil
	}

	rss, err := s.getResources()
	if err != nil {
		return err
	}

	log.Debugf("kubescaler: rss: unscheduledPods=%v", podNames(rss.unscheduledPods))

	failed, provisioning := s.checkWorkers(rss.workerList, currentTime)
	if len(failed) > 0 {
		// remove machines that are provisioning for a long time and with a not ready nodes
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
		if emptyNodes := getEmptyNodes(rss.readyNodes, rss.allPods); len(emptyNodes) > 0 {
			log.Debugf("kubescaler: scale up: there are %v ready empty nodes in the cluster", nodeNames(emptyNodes))
			return nil
		}

		if cfg.WorkersCountMax > 0 && cfg.WorkersCountMax > len(rss.workerList.Items) {
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
			log.Debugf("kubescaler: scaleup: workersCountMax(%d) >= number of workers(%d), skipping..",
				cfg.WorkersCountMax, len(rss.workerList.Items))
		}
	}

	if cfg.WorkersCountMin > 0 && cfg.WorkersCountMin < len(rss.workerList.Items) {
		if err = s.scaleDown(rss.scheduledPods, rss.workerList, cfg.IgnoredNodeLabels, currentTime); err != nil {
			return errors.Wrap(err, "scale down")
		}
	} else {
		log.Debugf("kubescaler: scaledown: workersCountMin(%d) < number of workers(%d), skipping..",
			cfg.WorkersCountMin, len(rss.workerList.Items))
	}

	return nil
}

type resources struct {
	allNodes        []*corev1.Node
	readyNodes      []*corev1.Node
	allPods         []*corev1.Pod
	scheduledPods   []*corev1.Pod
	unscheduledPods []*corev1.Pod
	workerList      *api.WorkerList
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

	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()

	workerList, err := s.workerManager.ListWorkers(context.Background())
	if err != nil {
		return nil, err
	}

	return &resources{
		allNodes:        allNodes,
		readyNodes:      filters.GetReadyNodes(allNodes),
		allPods:         allPods,
		scheduledPods:   filters.GetScheduledPods(allPods),
		unscheduledPods: filters.GetUnschedulablePods(allPods),
		workerList:      workerList,
	}, nil
}

func (s *Kubescaler) checkWorkers(workerList *api.WorkerList, currentTime time.Time) ([]string, []string) {
	//	provisioning machines:
	//	- state == 'pending' || 'running', https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html
	//	- running <= maxProvisionTime
	//	- nodeState != 'ready'
	//	- have no registered node, skip master
	//	- node state is not ready
	provisioning := make([]string, 0)

	//failedMachines:
	//	- running > maxProvisionTime
	//	- have no registered node, skip master
	failed := make([]string, 0)

	for _, worker := range workerList.Items {
		ignored := !(worker.MachineState == "pending" || worker.MachineState == "running") ||
			worker.NodeState == workers.NodeStateReady ||
			isMaster(worker)

		if ignored {
			continue
		}

		if worker.CreationTimestamp.Add(DefaultMaxMachineProvisionTime).After(currentTime) {
			provisioning = append(provisioning, worker.MachineID)
			continue
		}

		if worker.NodeName == "" {
			failed = append(failed, worker.MachineID)
		}
	}

	return failed, provisioning
}

func (s *Kubescaler) removeFailedMachines(ids []string) error {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()

	for _, id := range ids {
		if _, err := s.workerManager.DeleteWorker(context.Background(), "", id); err != nil {
			return err
		}

	}
	return nil
}

func (s *Kubescaler) machineTypes(permitted []string) []*provider.MachineType {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()

	out := make([]*provider.MachineType, 0, len(permitted))
	for _, name := range permitted {
		if mt := findMachine(name, s.workerManager.MachineTypes()); mt != nil {
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

func isMaster(w *api.Worker) bool {
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

// getConfigFile tries to locate the kubescaler config file.
// Sources priority order:
//   - file on the provided path;
//   - configmap;
//   - file on the default path.
//
// TODO: pass only configfile options
func getConfigFile(opts Options, cmGetter v1.ConfigMapsGetter) (persistentfile.Interface, error) {
	// try to use a file on provided path
	f, err := persistentfile.New(persistentfile.Config{
		Type: persistentfile.FSFile,
		Path: opts.ConfigFile,
		Perm: os.FileMode(0644),
	})
	if err == nil {
		return f, nil
	}

	// try to setup a configMap file
	f, err = persistentfile.New(persistentfile.Config{
		Type:               persistentfile.ConfigMapFile,
		ConfigMapName:      opts.ConfigMapName,
		ConfigMapNamespace: opts.ConfigMapNamespace,
		Key:                api.DefaultConfigMapKey,
		ConfigMapClient:    cmGetter,
	})
	if err == nil {
		return f, nil
	}

	// try to use a file on default path
	f, err = persistentfile.New(persistentfile.Config{
		Type: persistentfile.FSFile,
		Path: DefaultConfigFilepath,
		Perm: os.FileMode(0644),
	})
	if err == nil {
		return f, nil
	}

	return nil, errors.New("config file/configmap not found")
}

func (s *Kubescaler) MachineTypes() []*provider.MachineType {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.MachineTypes()
}

func (s *Kubescaler) CreateWorker(ctx context.Context, mtype string) (*api.Worker, error) {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.CreateWorker(ctx, mtype)
}

func (s *Kubescaler) GetWorker(ctx context.Context, id string) (*api.Worker, error) {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.GetWorker(ctx, id)
}

func (s *Kubescaler) ListWorkers(ctx context.Context) (*api.WorkerList, error) {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.ListWorkers(ctx)
}

func (s *Kubescaler) DeleteWorker(ctx context.Context, nodeName, id string) (*api.Worker, error) {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.DeleteWorker(ctx, nodeName, id)
}

func (s *Kubescaler) ReserveWorker(ctx context.Context, worker *api.Worker) (*api.Worker, error) {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.workerManager.ReserveWorker(ctx, worker)
}

func (s *Kubescaler) SetConfig(conf api.Config) error {
	// Recreate worker manager on config update
	if err := s.configManager.SetConfig(conf); err != nil {
		return err
	}

	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()

	return s.buildWorkerManager()
}

func (s *Kubescaler) GetConfig() api.Config {
	return s.configManager.GetConfig()
}

func (s *Kubescaler) PatchConfig(conf api.Config) error {
	if err := s.configManager.PatchConfig(conf); err != nil {
		return err
	}

	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()

	return s.buildWorkerManager()
}

func (s *Kubescaler) IsReady() bool {
	s.workerMutex.RLock()
	defer s.workerMutex.RUnlock()
	return s.isReady
}

func (s *Kubescaler) buildWorkerManager() error {
	cfg := s.configManager.GetConfig()

	vmProvider, err := factory.New(cfg.ClusterName, cfg.ProviderName, cfg.Provider)
	if err != nil {
		return errors.Wrapf(err, "build vm provider")
	}

	if cfg.SupergiantV1Config != nil {
		v, err := getServerVersion(s.kclient.RESTClient())
		if err != nil {
			return errors.Wrapf(err, "build kubernetes client")
		}
		cfg.SupergiantV1Config.KubeVersion = v.String()
	}
	userdata, err := buildUserdata(cfg)
	if err != nil {
		return errors.Wrap(err, "build userdata")
	}

	log.Infof("Create new worker manager for cluster %s", cfg.ClusterName)
	workerManager, err := workers.NewManager(cfg.ClusterName, s.kclient.Nodes(), vmProvider, userdata)
	if err != nil {
		return err
	}

	s.workerManager = workerManager
	s.isReady = true
	return nil
}

func buildUserdata(cfg api.Config) (string, error) {
	switch {
	case cfg.SupergiantV1Config != nil:
		cfg.SupergiantV1Config.ProviderName = cfg.ProviderName
		return parse(userDataTpl, cfg.SupergiantV1Config)
	case len(cfg.Userdata) > 0:
		return cfg.Userdata, nil
	}

	return "", errors.New("userdata configuration not found")
}

func parse(tpl string, data interface{}) (string, error) {
	t, err := template.New("userdata").Parse(tpl)
	if err != nil {
		return "", err
	}
	buff := &bytes.Buffer{}
	if err = t.Execute(buff, data); err != nil {
		return "", errors.Wrap(err, "parse supergiantV1 userdata")
	}
	return buff.String(), nil
}

// getServerVersion retrieves and parses the server's version (git version).
func getServerVersion(restclient rest.Interface) (*kubeversion.Info, error) {
	body, err := restclient.Get().AbsPath("/version").Do().Raw()
	if err != nil {
		return nil, err
	}
	var info kubeversion.Info
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("got '%s': %v", string(body), err)
	}
	return &info, nil
}
