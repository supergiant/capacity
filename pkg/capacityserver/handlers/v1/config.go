package v1

import (
	"encoding/json"
	"net/http"

	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/pkg/errors"
)

var (
	ErrInvalidPersistentConfig = errors.New("invalid persistent config")
)

type configHandler struct {
	pconf *capacity.PersistentConfig
}

func newConfigHandler(pconf *capacity.PersistentConfig) (*configHandler, error) {
	if pconf == nil {
		return nil, ErrInvalidPersistentConfig
	}
	return &configHandler{pconf}, nil
}

func (h *configHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(h.pconf.GetConfig()); err != nil {
		log.Errorf("handle: kubescaler: failed to encode config")
	}
}

func (h *configHandler) patchConfig(w http.ResponseWriter, r *http.Request) {
	patch := capacity.Config{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: decode: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.pconf.PatchConfig(&patch); err != nil {
		log.Errorf("handler: kubescaler: patch config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(h.pconf.GetConfig()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
