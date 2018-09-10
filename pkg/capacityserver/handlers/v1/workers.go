package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/log"
)

var (
	ErrInvalidWorkersManager = errors.New("invalid workers manager")
)

type workersHandler struct {
	m workers.WInterface
}

func newWorkersHandler(wiface workers.WInterface) (*workersHandler, error) {
	if wiface == nil {
		return nil, ErrInvalidWorkersManager
	}
	return &workersHandler{wiface}, nil
}

func (h *workersHandler) listMachineTypes(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/machinetypes workers listMachineTypes
	//
	// Lists all provider's machine types.
	//
	// This will show all provider's machine types.
	//
	//     Produces:
	//     - application/json
	//
	//     Responses:
	//     200: machineTypesListResponse

	if err := json.NewEncoder(w).Encode(h.m.MachineTypes()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *workersHandler) createWorker(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/workers workers createWorker
	//
	// Create a new worker with the specified machine type.
	//
	// This will create a new worker.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: https, http
	//
	//     Responses:
	//     201: workerResponse

	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	worker, err = h.m.CreateWorker(r.Context(), worker.MachineType)
	if err != nil {
		log.Errorf("handler: kubescaler: create worker: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("handler: kubescaler: %s worker (%s) has been created ", worker.MachineID, worker.MachineType)

	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: create %s worker: failed to write response: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *workersHandler) getWorker(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/workers/{machineID} workers getWorker
	//
	// Get a worker with the specified machineID.
	//
	// This will get a worker.
	//
	//     Produces:
	//     - application/json
	//
	//     Responses:
	//     200: workerResponse

	vars := mux.Vars(r)
	if vars == nil {
		log.Errorf("handler: kubescaler: delete worker: vars wasn't found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error
	worker := &workers.Worker{}

	worker, err = h.m.GetWorker(r.Context(), vars["machineID"])
	if err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("handler: kubescaler: %s worker has been deleted", worker.MachineID)

	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: failed to write response: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *workersHandler) listWorkers(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/workers workers listWorkers
	//
	// Lists all workers.
	//
	// This will show all workers.
	//
	//     Produces:
	//     - application/json
	//
	//     Responses:
	//     200: workerListResponse

	workers, err := h.m.ListWorkers(r.Context())
	if err != nil {
		log.Errorf("handler: kubescaler: list workers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(workers); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *workersHandler) updateWorker(w http.ResponseWriter, r *http.Request) {
	// swagger:route PATCH /api/v1/workers/{machineID} workers updateWorker
	//
	// Update a worker.
	//
	// This will update a worker.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Responses:
	//     200: workerResponse

	vars := mux.Vars(r)
	if vars == nil {
		log.Errorf("handler: kubescaler: delete worker: vars wasn't found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(&worker); err != nil {
		log.Errorf("handler: kubescaler: patch worker: decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	worker.MachineID = vars["machineID"]
	worker, err = h.m.ReserveWorker(r.Context(), worker)
	if err != nil {
		if errors.Cause(err) == workers.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		log.Errorf("handler: kubescaler: patch worker: reserve id=%s, node=%s: %v", worker.MachineID, worker.NodeName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: patch %s worker: failed to write response: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *workersHandler) deleteWorker(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/workers/{machineID} workers deleteWorker
	//
	// Delete a worker with the specified machineID.
	//
	// This will delete a worker.
	//
	//     Produces:
	//     - application/json
	//
	//     Responses:
	//     200: workerResponse

	vars := mux.Vars(r)
	if vars == nil {
		log.Errorf("handler: kubescaler: delete worker: vars wasn't found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error
	worker := &workers.Worker{}

	worker, err = h.m.DeleteWorker(r.Context(), "", vars["machineID"])
	if err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("handler: kubescaler: %s worker has been deleted", worker.MachineID)

	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: failed to write response: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
