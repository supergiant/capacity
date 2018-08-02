package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/supergiant/capacity/pkg/capacityserver/handlers/v1"
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/version"
	"github.com/supergiant/capacity/pkg/kubescaler"
)

func Router(ks *capacity.Kubescaler) (*mux.Router, error) {
	handlerV1, err := v1.New(ks)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter()
	r.Path("/version").Methods(http.MethodGet).HandlerFunc(version.Handler)

	apiv1 := r.PathPrefix("/api/v1").Subrouter()
	handlerV1.RegisterTo(apiv1)

	return r, nil
}
