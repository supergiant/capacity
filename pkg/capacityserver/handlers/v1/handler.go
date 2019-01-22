package v1

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"fmt"
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

func (h *HandlerV1) RegisterTo(ks *kubescaler.Kubescaler, r *mux.Router) {
	r.Path("/config").Methods(http.MethodPost).HandlerFunc(h.config.createConfig)

	r.Path("/config").Methods(http.MethodGet).HandlerFunc(readyMiddleware(ks,
		h.config.getConfig))
	r.Path("/config").Methods(http.MethodPatch).HandlerFunc(readyMiddleware(ks,
		h.config.patchConfig))

	r.Path("/machinetypes").HandlerFunc(readyMiddleware(ks, h.workers.listMachineTypes)).
		Methods(http.MethodGet)

	r.Path("/workers").Methods(http.MethodPost).
		HandlerFunc(readyMiddleware(ks, h.workers.createWorker))
	r.Path("/workers").Methods(http.MethodGet).
		HandlerFunc(readyMiddleware(ks, h.workers.listWorkers))
	r.Path("/workers/{machineID}").Methods(http.MethodGet).
		HandlerFunc(readyMiddleware(ks, h.workers.getWorker))
	r.Path("/workers/{machineID}").Methods(http.MethodPatch).
		HandlerFunc(readyMiddleware(ks, h.workers.updateWorker))
	r.Path("/workers/{machineID}").Methods(http.MethodDelete).
		HandlerFunc(readyMiddleware(ks, h.workers.deleteWorker))
}

// We allow all method only when kubescaler was configured
func readyMiddleware(ks *kubescaler.Kubescaler, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ks.IsReady() {
			h.ServeHTTP(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "kube scaler was not configured yet, to configure "+
			"make POST request to /api/v1/config with valid config object")
	}
}
