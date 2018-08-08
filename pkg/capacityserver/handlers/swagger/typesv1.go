package swagger

import (
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/provider"
)

// configResponse contains an application config parameters.
// swagger:response configResponse
type configResponse struct {
	// in:body
	Config *capacity.Config `json:"config"`
}

// machineTypesListResponse contains a list of workers.
// swagger:response machineTypesListResponse
type machineTypesListResponse struct {
	// in:body
	MachineTypes []*provider.MachineType `json:"machineTypes"`
}

// workerResponse contains a worker representation.
// swagger:response workerResponse
type workerResponse struct {
	// in:body
	Worker *workers.Worker `json:"worker"`
}

// workerListResponse contains a list of workers.
// swagger:response workerListResponse
type workerListResponse struct {
	// in:body
	WorkerList *workers.WorkerList `json:"workerList"`
}

// machineIDParam is used to identify a worker.
// swagger:parameters deleteWorker
type machineIDParam struct {
	// in:path
	// required: true
	MachineID string `json:"machineID"`
}
