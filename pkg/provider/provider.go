package provider

import (
	"context"
	"sort"
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

// TODO: split string and resource representation
type MachineType struct {
	Name           string            `json:"name"`
	Memory         string            `json:"memory"`
	CPU            string            `json:"cpu"`
	MemoryResource resource.Quantity `json:"-"`
	CPUResource    resource.Quantity `json:"-"`
}

func SortedMachineTypes(mtypes []*MachineType) []*MachineType {
	sorted := make([]*MachineType, len(mtypes))
	copy(sorted, mtypes)
	sort.SliceStable(sorted, func(i, j int) bool {
		byCPU := sorted[j].CPUResource.Cmp(sorted[i].CPUResource) > -1
		byMemory := sorted[j].MemoryResource.Cmp(sorted[i].MemoryResource) > -1
		return byCPU && byMemory
	})
	return sorted
}

type Config map[string]string

type Provider interface {
	Name() string
	GetMachineID(providerID string) (string, error)
	MachineTypes(ctx context.Context) ([]*MachineType, error)
	Machines(ctx context.Context) ([]*Machine, error)
	CreateMachine(ctx context.Context, name, mtype, clusterRole, userData string, config Config) (*Machine, error)
	DeleteMachine(ctx context.Context, id string) (*Machine, error)
}
