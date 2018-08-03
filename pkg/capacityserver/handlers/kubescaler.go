package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/kubescaler/workers"
	"github.com/supergiant/capacity/pkg/log"
)

type kubescalerHandlerV1 struct {
	ks *capacity.Kubescaler
}

func (h *kubescalerHandlerV1) register(r *mux.Router) {

	ks := r.PathPrefix("/kubescaler").Subrouter()

	ks.Path("/workers").Methods(http.MethodPost).HandlerFunc(h.createWorker)
	ks.Path("/workers").Methods(http.MethodGet).HandlerFunc(h.listWorkers)
	ks.Path("/workers").Methods(http.MethodDelete).HandlerFunc(h.deleteWorker)

	ks.Path("/config").Methods(http.MethodGet).HandlerFunc(h.getConfig)
	ks.Path("/config").Methods(http.MethodPatch).HandlerFunc(h.patchConfig)
}

func (h *kubescalerHandlerV1) getConfig(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(h.ks.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: failed to encode config")
	}
}

func (h *kubescalerHandlerV1) patchConfig(w http.ResponseWriter, r *http.Request) {
	patch := capacity.Config{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.ks.PatchConfig(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(h.ks.GetConfig()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *kubescalerHandlerV1) createWorker(w http.ResponseWriter, r *http.Request) {
	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	worker, err = h.ks.CreateWorker(r.Context(), worker.MachineType)
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

func (h *kubescalerHandlerV1) listWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.ks.ListWorkers(r.Context())
	if err != nil {
		log.Errorf("handler: kubescaler: list workers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(workers); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *kubescalerHandlerV1) deleteWorker(w http.ResponseWriter, r *http.Request) {
	var err error
	worker := &workers.Worker{}
	if err = json.NewDecoder(r.Body).Decode(worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	worker, err = h.ks.DeleteWorker(r.Context(), worker.NodeName, worker.MachineID)
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
