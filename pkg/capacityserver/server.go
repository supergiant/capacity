package capacityserver

import (
	"net/http"
	"time"

	"github.com/rs/cors"

	"github.com/supergiant/capacity/pkg/capacityserver/handlers"
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
)

type Config struct {
	KubescalerConfig string
	KubeConfig       string
	ListenAddr       string
}

type API struct {
	ks  *capacity.Kubescaler
	srv http.Server
}

func New(conf Config) (*API, error) {
	ks, err := capacity.New(conf.KubeConfig, conf.KubescalerConfig)
	if err != nil {
		return nil, err
	}

	handler, err := handlers.Router(ks)
	if err != nil {
		return nil, err
	}

	return &API{
		ks: ks,
		srv: http.Server{
			Addr:         conf.ListenAddr,
			Handler:      cors.Default().Handler(handler),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}, nil
}

func (a *API) Start(stopCh <-chan struct{}) error {
	log.Infof("capacityservice: listen on %q", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *API) Shutdown() error {
	return nil
}
