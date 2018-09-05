package listers

import (
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	v1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Registry is a registry providing listers to list all pods or nodes.
type Registry interface {
	AllNodeLister() NodeLister
	AllPodLister() PodLister
}

type RegistryImpl struct {
	allNodeLister NodeLister
	allPodLister  PodLister
}

// NewRegistry returns a registry providing listers to list all pods or nodes.
func NewRegistry(allNode NodeLister, allPod PodLister) Registry {
	return RegistryImpl{
		allNodeLister: allNode,
		allPodLister:  allPod,
	}
}

// NewRegistryWithDefaultListers returns a registry filled with listers of the default implementations.
func NewRegistryWithDefaultListers(kubeClient kubernetes.Interface, stopChannel <-chan struct{}) Registry {
	return NewRegistry(NewAllNodeLister(kubeClient, stopChannel), NewAllPodLister(kubeClient, stopChannel))
}

// AllNodeLister returns the AllNodeLister registered to this registry.
func (r RegistryImpl) AllNodeLister() NodeLister {
	return r.allNodeLister
}

// AllPodLister returns the AllPodLister registered to this registry.
func (r RegistryImpl) AllPodLister() PodLister {
	return r.allPodLister
}

// PodLister lists pods.
type PodLister interface {
	List() ([]*apiv1.Pod, error)
}

// UnschedulablePodLister lists all pods.
type AllPodLister struct {
	podLister v1lister.PodLister
}

// List returns all pods.
func (allPodLister *AllPodLister) List() ([]*apiv1.Pod, error) {
	return allPodLister.podLister.List(labels.Everything())
}

// NewAllPodLister returns a lister providing all pods.
func NewAllPodLister(kubeClient kubernetes.Interface, stopchannel <-chan struct{}) PodLister {
	return NewAllPodInNamespaceLister(kubeClient, apiv1.NamespaceAll, stopchannel)
}

// NewAllPodInNamespaceLister returns a lister providing all pods.
func NewAllPodInNamespaceLister(kubeClient kubernetes.Interface, namespace string, stopchannel <-chan struct{}) PodLister {
	podListWatch := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", namespace, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	podLister := v1lister.NewPodLister(store)
	podReflector := cache.NewReflector(podListWatch, &apiv1.Pod{}, store, time.Hour)
	go podReflector.Run(stopchannel)
	return &AllPodLister{
		podLister: podLister,
	}
}

// NodeLister lists nodes.
type NodeLister interface {
	List() ([]*apiv1.Node, error)
}

// AllNodeLister lists all nodes
type AllNodeLister struct {
	nodeLister v1lister.NodeLister
}

// List returns all nodes
func (allNodeLister *AllNodeLister) List() ([]*apiv1.Node, error) {
	nodes, err := allNodeLister.nodeLister.List(labels.Everything())
	if err != nil {
		return []*apiv1.Node{}, err
	}
	allNodes := append(make([]*apiv1.Node, 0, len(nodes)), nodes...)
	return allNodes, nil
}

// NewAllNodeLister builds a node lister that returns all nodes (ready and unready)
func NewAllNodeLister(kubeClient kubernetes.Interface, stopchannel <-chan struct{}) NodeLister {
	listWatcher := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "nodes", apiv1.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	nodeLister := v1lister.NewNodeLister(store)
	reflector := cache.NewReflector(listWatcher, &apiv1.Node{}, store, time.Hour)
	go reflector.Run(stopchannel)
	return &AllNodeLister{
		nodeLister: nodeLister,
	}
}
