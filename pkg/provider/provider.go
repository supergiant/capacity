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

type Machine struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
	State             string    `json:"state"`
}

type MachineType struct {
	Name   string            `json:"name"`
	CPU    resource.Quantity `json:"cpu"`
	Memory resource.Quantity `json:"memory"`
}

type Config map[string]string

type Provider interface {
	Name() string
	MachineTypes(ctx context.Context) ([]*MachineType, error)
	Machines(ctx context.Context) ([]*Machine, error)
	CreateMachine(ctx context.Context, name, mtype, clusterRole, userData string, config Config) (*Machine, error)
	DeleteMachine(ctx context.Context, id string) (*Machine, error)
}
