package config

import (
	"errors"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrHostNotSpecified   = errors.New("host not specified")
	ErrInvalidCredentials = errors.New("invalid credentials")
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

// GetBasicAuthConfig is a helper function that builds configs for a kubernetes client
// that uses a basic authentication.
// https://kubernetes.io/docs/admin/authentication/#static-password-file
func GetBasicAuthConfig(host, port, username, pass string) (*rest.Config, error) {
	if host == "" {
		return nil, ErrHostNotSpecified
	}
	if username == "" || pass == "" {
		return nil, ErrInvalidCredentials
	}

	if port != "" {
		host += ":" + port
	}

	return &rest.Config{
		Host:     host,
		Username: username,
		Password: pass,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}, nil
}

func GetKubernetesClientSetBasicAuth(host, port, username, pass string) (*kubernetes.Clientset, error) {
	config, err := GetBasicAuthConfig(host, port, username, pass)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
