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
	ID        string
	Name      string
	Type      string
	CreatedAt time.Time
	State     string
}

type MachineType struct {
	Name   string
	CPU    resource.Quantity
	Memory resource.Quantity
}

type Config map[string]string

type Provider interface {
	Name() string
	Machines(ctx context.Context) ([]*Machine, error)
	AvailableMachineTypes(ctx context.Context) ([]*MachineType, error)
	CreateMachine(ctx context.Context, name, mtype, clusterRole, userData string, config Config) (*Machine, error)
	DeleteMachine(ctx context.Context, id string) (*Machine, error)
}
