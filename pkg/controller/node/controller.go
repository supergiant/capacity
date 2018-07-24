package node

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
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
// Controller implementation logic for Node resources goes here.

func (bc *NodeController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on node.NodeController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:informers:group=core,version=v1,kind=Node
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;watch;list
// +kubebuilder:controller:group=core,version=v1,kind=Node,resource=nodes
type NodeController struct {
	// INSERT ADDITIONAL FIELDS HERE
	nodeLister corev1lister.NodeLister
	nodeclient corev1client.CoreV1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	noderecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &NodeController{
		nodeLister:   arguments.ControllerManager.GetInformerProvider(&corev1.Node{}).(corev1informer.NodeInformer).Lister(),
		nodeclient:   arguments.KubernetesClientSet.CoreV1(),
		noderecorder: arguments.CreateRecorder("NodeController"),
	}

	// Create a new controller that will call NodeController.Reconcile on changes to Nodes
	gc := &controller.GenericController{
		Name:             "NodeController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&corev1.Node{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a Node Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the NodeController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	return gc, nil
}
