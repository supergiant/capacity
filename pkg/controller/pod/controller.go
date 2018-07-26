package pod

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	corev1 "k8s.io/api/core/v1"
	corev1informer "k8s.io/client-go/informers/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/supergiant/capacity/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for Pod resources goes here.

func (bc *PodController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on pod.PodController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:informers:group=core,version=v1,kind=Pod
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;watch;list
// +kubebuilder:controller:group=core,version=v1,kind=Pod,resource=pods
type PodController struct {
	// INSERT ADDITIONAL FIELDS HERE
	podLister corev1lister.PodLister
	podclient corev1client.CoreV1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	podrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &PodController{
		podLister:   arguments.ControllerManager.GetInformerProvider(&corev1.Pod{}).(corev1informer.PodInformer).Lister(),
		podclient:   arguments.KubernetesClientSet.CoreV1(),
		podrecorder: arguments.CreateRecorder("PodController"),
	}

	// Create a new controller that will call PodController.Reconcile on changes to Pods
	gc := &controller.GenericController{
		Name:             "PodController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&corev1.Pod{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a Pod Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the PodController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	gc.WatchControllerOf(&corev1.Node{}, eventhandlers.Path{})

	return gc, nil
}
