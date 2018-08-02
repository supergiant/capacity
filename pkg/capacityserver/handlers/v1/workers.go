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
	m *workers.Manager
}

func newWorkersHandler(pconf *workers.Manager) (*workersHandler, error) {
	if pconf == nil {
		return nil, ErrInvalidWorkersManager
	}
	return &workersHandler{pconf}, nil
}

func (h *workersHandler) createWorker(w http.ResponseWriter, r *http.Request) {
	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	worker, err = h.m.CreateWorker(r.Context(), worker.MachineType)
	if err != nil {
		log.Errorf("handler: kubescaler: create worker: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("handler: kubescaler: %s worker (%s) has been created ", worker.MachineID, worker.MachineType)

	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: create %s worker: failed to write response: %v", worker.MachineID, err)
		return
	}
}

func (h *workersHandler) listWorkers(w http.ResponseWriter, r *http.Request) {
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

func (h *workersHandler) deleteWorker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		log.Errorf("handler: kubescaler: delete worker: vars wasn't found")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(worker); err != nil {
		log.Errorf("handler: kubescaler: delete worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	worker, err = h.m.DeleteWorker(r.Context(), worker.NodeName, worker.MachineID)
	if err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("handler: kubescaler: %s worker has been deleted", worker.MachineID)

	if err = json.NewEncoder(w).Encode(worker); err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: failed to write response: %v", worker.MachineID, err)
		return
	}
}
