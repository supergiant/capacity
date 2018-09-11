package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	_ "github.com/supergiant/capacity/pkg/capacityserver/handlers/swagger" // for swagger generation
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/v1"
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/version"
	"github.com/supergiant/capacity/pkg/kubescaler"
)

func Handler(ks *capacity.Kubescaler) (*mux.Router, error) {
	handlerV1, err := v1.New(ks)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter()

	r.Path("/version").Methods(http.MethodGet).HandlerFunc(version.Handler)

	apiv1 := r.PathPrefix("/api/v1").Subrouter()
	handlerV1.RegisterTo(apiv1)
	apiv1.Use(
		mux.MiddlewareFunc(setContentType),
	)
	r.Use(
		cors.AllowAll().Handler,
	)

	return r, nil
}

func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
