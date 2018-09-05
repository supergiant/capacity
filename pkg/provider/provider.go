package provider

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Provider specific tags:
const (
	TagCluster = "KubernetesCluster"
)

// Separators for custom lists and maps:
// list: "val1,val2"
// map:  "key1=val1,key2=val2"
const (
	ListSep   = ","
	KeyValSep = "="
)

// Machine is an existing, running machine on the cloud platform, and
// it contains basic information needed to allow the capacity
// controller to make informed decisions.
type Machine struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
	State             string    `json:"state"`
}

// TODO: split string and resource representation
// MachineType contains information about the specific resources and
// identities of available machines on the respective coud platform.
type MachineType struct {
	Name           string            `json:"name"`
	Memory         string            `json:"memory"`
	CPU            string            `json:"cpu"`
	MemoryResource resource.Quantity `json:"-"`
	CPUResource    resource.Quantity `json:"-"`
}

type Config map[string]string

// Provider contains everything needed in order to see existing
// infrastructure and create or delete machines of various types.
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// GetMachineID extracts the awsInstanceID from the providerID
	//
	// providerID represents the id for an instance in the kubernetes API;
	// the following form
	//  * aws:///<zone>/<awsInstanceId>
	//  * aws:////<awsInstanceId>
	//  * <awsInstanceId>
	GetMachineID(providerID string) (string, error)

	// MachineTypes contacts AWS to ask what EC2 instance types
	// are allowed to be created. It returns a slice of objects with the
	// name (e.g. "m4.large", RAM, and CPU of each type).
	MachineTypes(ctx context.Context) ([]*MachineType, error)

	// Machines contacts the AWS API to get a list of the current
	// instances present in the region.
	Machines(ctx context.Context) ([]*Machine, error)

	// CreateMachine takes a whole bunch of information, including a
	// machine config, and attempts to create an instance using the AWS
	// account it has access to. If successful, the machine information
	// is returned.
	CreateMachine(ctx context.Context, name, mtype, clusterRole, userData string, config Config) (*Machine, error)

	// DeleteMachine sends a request to AWS to delete the instance passed
	// and returns the instance's id and state (if successful).
	DeleteMachine(ctx context.Context, id string) (*Machine, error)
}
