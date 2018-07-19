package inject

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	capacityv1 "github.com/supergiant/capacity/pkg/apis/capacity/v1"
	rscheme "github.com/supergiant/capacity/pkg/client/clientset/versioned/scheme"
	"github.com/supergiant/capacity/pkg/controller/clustercapacity"
	"github.com/supergiant/capacity/pkg/controller/node"
	"github.com/supergiant/capacity/pkg/controller/pod"
	"github.com/supergiant/capacity/pkg/inject/args"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	rscheme.AddToScheme(scheme.Scheme)

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		if err := arguments.ControllerManager.AddInformerProvider(&capacityv1.ClusterCapacity{}, arguments.Informers.Capacity().V1().ClusterCapacities()); err != nil {
			return err
		}

		// Add Kubernetes informers
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Node{}, arguments.KubernetesInformers.Core().V1().Nodes()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Pod{}, arguments.KubernetesInformers.Core().V1().Pods()); err != nil {
			return err
		}

		if c, err := clustercapacity.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		if c, err := node.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		if c, err := pod.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	Injector.CRDs = append(Injector.CRDs, &capacityv1.ClusterCapacityCRD)
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"capacity.supergiant.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"nodes",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"pods",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	// Inject GroupVersions
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "capacity.supergiant.io",
		Version: "v1",
	})
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
