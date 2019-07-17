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
	PriceHour      int64             `json:"priceHour"`
	Description    string            `json:"description"`
}

func SortedMachineTypes(mtypes []*MachineType) []*MachineType {
	sorted := make([]*MachineType, len(mtypes))
	copy(sorted, mtypes)
	sort.Slice(sorted, func(i, j int) bool {
		equalPrice := sorted[i].PriceHour == sorted[j].PriceHour
		lessPrice := sorted[i].PriceHour < sorted[j].PriceHour
		moreCPU := sorted[i].CPUResource.Cmp(sorted[j].CPUResource) == 1
		equalCPU := sorted[i].CPUResource.Cmp(sorted[j].CPUResource) == 0
		moreMemory := sorted[i].MemoryResource.Cmp(sorted[j].MemoryResource) == 1

		if equalPrice {
			if equalCPU {
				return moreMemory
			}
			return moreCPU
		}
		return lessPrice
	})
	return sorted
}

type Config map[string]string

type Provider interface {
	Name() string
	ParseMachineID(providerID string) (string, error)
	GetMachine(ctx context.Context, id string) (*Machine, error)
	MachineTypes(ctx context.Context) ([]*MachineType, error)
	Machines(ctx context.Context) ([]*Machine, error)
	CreateMachine(ctx context.Context, name, mtype, clusterRole, userData string, config Config) (*Machine, error)
	DeleteMachine(ctx context.Context, id string) (*Machine, error)
}
