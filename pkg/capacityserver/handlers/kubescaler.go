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

func (k *kubescalerHandlerV1) register(r *mux.Router) {

	ks := r.PathPrefix("/kubescaler").Subrouter()

	ks.Path("/workers").Methods(http.MethodPost).HandlerFunc(k.createWorker)
	ks.Path("/workers").Methods(http.MethodGet).HandlerFunc(k.listWorkers)
	ks.Path("/workers").Methods(http.MethodDelete).HandlerFunc(k.deleteWorker)

	ks.Path("/config").Methods(http.MethodGet).HandlerFunc(k.getConfig)
	ks.Path("/config").Methods(http.MethodPatch).HandlerFunc(k.patchConfig)
}

func (k *kubescalerHandlerV1) getConfig(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(k.ks.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: failed to encode config")
	}
}

func (k *kubescalerHandlerV1) patchConfig(w http.ResponseWriter, r *http.Request) {
	patch := capacity.Config{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := k.ks.PatchConfig(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(k.ks.GetConfig()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (k *kubescalerHandlerV1) createWorker(w http.ResponseWriter, r *http.Request) {
	worker := workers.Worker{}
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := k.ks.CreateWorker(r.Context(), worker.MachineType); err != nil {
		log.Errorf("handler: kubescaler: create worker: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof("handler: kubescaler: create worker with %s machine type", worker.MachineType)
	w.WriteHeader(http.StatusAccepted)
}

func (k *kubescalerHandlerV1) listWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := k.ks.ListWorkers(r.Context())
	if err != nil {
		log.Errorf("handler: kubescaler: list workers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(workers); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (k *kubescalerHandlerV1) deleteWorker(w http.ResponseWriter, r *http.Request) {
	worker := workers.Worker{}
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		log.Errorf("handler: kubescaler: create worker: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := k.ks.DeleteWorker(r.Context(), worker.NodeName, worker.MachineID); err != nil {
		log.Errorf("handler: kubescaler: delete %s worker: %v", worker.MachineID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof("handler: kubescaler: %s worker has been deleted", worker.MachineID)
}
