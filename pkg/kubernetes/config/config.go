package config

import (
	"errors"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
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

// GetCoreV1Client creates a client for talking to a Kubernetes corev1 resources.
func GetCoreV1Client(masterURL, kubeconfig string) (corev1client.CoreV1Interface, error) {
	config, err := GetConfig(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}

	return corev1client.NewForConfig(config)
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
