package handlers

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/supergiant/capacity/pkg/kubescaler"
)

var (
	ErrNoKubescaler = errors.New("kubescaler should be provided")
)

func New(ks *capacity.Kubescaler) (*mux.Router, error) {
	if ks == nil {
		return nil, ErrNoKubescaler
	}

	r := mux.NewRouter()

	r.Path("/status").Methods(http.MethodGet).HandlerFunc(getStatus)

	ksHanler := kubescalerHandlerV1{ks}
	ksHanler.register(r)

	return r, nil
}
