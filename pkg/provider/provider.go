package provider

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Machine struct {
	ID        string
	Type      string
	CreatedAt time.Time
}

type MachineSize struct {
	Type   string
	CPU    resource.Quantity
	Memory resource.Quantity
}

type Provider interface {
	Name() string
	Machines() ([]*Machine, error)
	AvailableMachineTypes() ([]MachineSize)
	CreateMachine(*Machine) (*Machine, error)
	DeleteMachine(string) error
}
