package swagger

import (
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
)

// configResponse contains an application config parameters.
// swagger:response configResponse
type configResponse struct {
	// in:body
	Config *capacity.Config `json:"config"`
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
