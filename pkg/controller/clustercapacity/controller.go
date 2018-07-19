package clustercapacity

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/client-go/tools/record"

	capacityv1 "github.com/supergiant/capacity/pkg/apis/capacity/v1"
	capacityv1client "github.com/supergiant/capacity/pkg/client/clientset/versioned/typed/capacity/v1"
	capacityv1informer "github.com/supergiant/capacity/pkg/client/informers/externalversions/capacity/v1"
	capacityv1lister "github.com/supergiant/capacity/pkg/client/listers/capacity/v1"

	"github.com/supergiant/capacity/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for ClusterCapacity resources goes here.

func (bc *ClusterCapacityController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on clustercapacity.ClusterCapacityController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:controller:group=capacity,version=v1,kind=ClusterCapacity,resource=clustercapacities
type ClusterCapacityController struct {
	// INSERT ADDITIONAL FIELDS HERE
	clustercapacityLister capacityv1lister.ClusterCapacityLister
	clustercapacityclient capacityv1client.CapacityV1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	clustercapacityrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &ClusterCapacityController{
		clustercapacityLister: arguments.ControllerManager.GetInformerProvider(&capacityv1.ClusterCapacity{}).(capacityv1informer.ClusterCapacityInformer).Lister(),

		clustercapacityclient:   arguments.Clientset.CapacityV1(),
		clustercapacityrecorder: arguments.CreateRecorder("ClusterCapacityController"),
	}

	// Create a new controller that will call ClusterCapacityController.Reconcile on changes to ClusterCapacitys
	gc := &controller.GenericController{
		Name:             "ClusterCapacityController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&capacityv1.ClusterCapacity{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a ClusterCapacity Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the ClusterCapacityController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	return gc, nil
}
