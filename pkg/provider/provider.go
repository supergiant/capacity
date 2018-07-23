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

type MachineType struct {
	Name   string
	CPU    resource.Quantity
	Memory resource.Quantity
}

type Provider interface {
	Name() string
	Machines() ([]*Machine, error)
	AvailableMachineTypes() ([]MachineType)
	CreateMachine(string) (*Machine, error)
	DeleteMachine(string) error
}
