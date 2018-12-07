package swagger

import (
	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/provider"
)

// configResponse contains an application config parameters.
// swagger:response configResponse
type configResponse struct {
	// in:body
	Config *api.Config `json:"config"`
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
	Worker *api.Worker `json:"worker"`
}

// workerListResponse contains a list of workers.
// swagger:response workerListResponse
type workerListResponse struct {
	// in:body
	WorkerList *api.WorkerList `json:"workerList"`
}

// machineIDParam is used to identify a worker.
// swagger:parameters getWorker updateWorker deleteWorker
type machineIDParam struct {
	// in:path
	// required: true
	MachineID string `json:"machineID"`
}
