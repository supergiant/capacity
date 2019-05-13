package instancetypes

import (
	"fmt"
	"sync"
)

//go:generate go run gen/main.go

type manager struct {
	vmu       sync.RWMutex
	regionVMs map[string][]VM
}

type VM struct {
	Name      string
	VCPU      string
	MemoryGiB string
	GPU       string
}

func (s manager) regionTypes(region string) ([]VM, error) {
	s.vmu.RLock()
	defer s.vmu.RUnlock()

	if s.regionVMs == nil {
		return nil, fmt.Errorf("regionVMs is nil")
	}

	if _, ok := s.regionVMs[region]; !ok {
		return nil, fmt.Errorf("unknown region: %s", region)
	}
	return s.regionVMs[region], nil
}
