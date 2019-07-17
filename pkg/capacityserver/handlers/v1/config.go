package v1

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/api"
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
)

var (
	ErrInvalidPersistentConfig = errors.New("invalid persistent configHandler")
)

type ConfigManager interface {
	GetConfig() api.Config
	SetConfig(api.Config) error
	PatchConfig(api.Config) error
}

type configHandler struct {
	cm ConfigManager
}

func newConfigHandler(pconf *kubescaler.Kubescaler) (*configHandler, error) {
	if pconf == nil {
		return nil, ErrInvalidPersistentConfig
	}
	return &configHandler{pconf}, nil
}

func (h *configHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/config config getConfig
	//
	// Returns a current view of the kubescaler configuration.
	//
	// This will show all configuration parameters of the application.
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: https, http
	//
	//     Responses:
	//     200: configResponse

	if err := json.NewEncoder(w).Encode(h.cm.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: get config: failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *configHandler) patchConfig(w http.ResponseWriter, r *http.Request) {
	// swagger:route PATCH /api/v1/config config updateConfig
	//
	// Returns a new view of the kubescaler configuration.
	//
	// This will update current configuration of the application.
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
	//     200: configResponse

	patch := api.Config{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.cm.PatchConfig(patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(h.cm.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: patch config: failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *configHandler) createConfig(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/config config createConfig
	//
	// Returns a view of the kubescaler configuration.
	//
	// This will set configuration for the application.
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
	//     201: configResponse
	log.Info("Create config")

	cfg := api.Config{}
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		log.Errorf("handler: kubescaler: cfg config: decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("Set config")
	if err := h.cm.SetConfig(cfg); err != nil {
		log.Errorf("handler: kubescaler: create config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(h.cm.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: get config: failed to encode")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
