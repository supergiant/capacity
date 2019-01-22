package v1

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/kubescaler"
)

var (
	ErrNoKubescaler = errors.New("kubescaler should be provided")
)

type HandlerV1 struct {
	workers *workersHandler
	config  *configHandler
}

func New(ks *kubescaler.Kubescaler) (*HandlerV1, error) {
	if ks == nil {
		return nil, ErrNoKubescaler
	}
	wh, err := newWorkersHandler(ks)
	if err != nil {
		return nil, err
	}
	cf, err := newConfigHandler(ks)
	if err != nil {
		return nil, err
	}

	return &HandlerV1{
		workers: wh,
		config:  cf,
	}, nil
}

func (h *HandlerV1) RegisterTo(r *mux.Router) {
	r.Path("/config").Methods(http.MethodGet).HandlerFunc(h.config.getConfig)
	r.Path("/config").Methods(http.MethodPatch).HandlerFunc(h.config.patchConfig)
	r.Path("/config").Methods(http.MethodPatch).HandlerFunc(h.config.createConfig)

	r.Path("/machinetypes").HandlerFunc(h.workers.listMachineTypes).Methods(http.MethodGet)

	r.Path("/workers").Methods(http.MethodPost).HandlerFunc(h.workers.createWorker)
	r.Path("/workers").Methods(http.MethodGet).HandlerFunc(h.workers.listWorkers)
	r.Path("/workers/{machineID}").Methods(http.MethodGet).HandlerFunc(h.workers.getWorker)
	r.Path("/workers/{machineID}").Methods(http.MethodPatch).HandlerFunc(h.workers.updateWorker)
	r.Path("/workers/{machineID}").Methods(http.MethodDelete).HandlerFunc(h.workers.deleteWorker)
}
