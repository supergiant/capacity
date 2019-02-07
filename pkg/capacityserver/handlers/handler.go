package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	_ "github.com/supergiant/capacity/pkg/capacityserver/handlers/swagger" // for swagger generation
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/v1"
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/version" //"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/kubescaler"
)

func RegisterRouter(ks *kubescaler.Kubescaler, handler *v1.HandlerV1) (*mux.Router, error) {
	r := mux.NewRouter()

	r.Path("/version").Methods(http.MethodGet).HandlerFunc(version.Handler)

	apiv1 := r.PathPrefix("/api/v1").Subrouter()
	handler.RegisterTo(ks, apiv1)
	apiv1.Use(
		mux.MiddlewareFunc(setContentType),
	)

	return r, nil
}

func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
