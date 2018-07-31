package config

import (
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig creates a *rest.Config for talking to a Kubernetes apiserver.
// If kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetConfig(masterURL, kubeconfig string) (*rest.Config, error) {
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	}
	return rest.InClusterConfig()
}

// GetKubernetesClientSet creates a *kubernetes.ClientSet for talking to a Kubernetes apiserver.
// If kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetKubernetesClientSet(masterURL, kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := GetConfig(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// GetKubernetesInformers creates a informers.SharedInformerFactory for talking to a Kubernetes apiserver.
// If kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetKubernetesInformers(masterURL, kubeconfig string) (informers.SharedInformerFactory, error) {
	config, err := GetConfig(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}
	i, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return informers.NewSharedInformerFactory(i, time.Minute*5), nil
}
