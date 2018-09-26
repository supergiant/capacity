package capacityserver

import (
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/capacityserver/handlers"
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/v1"
	kubescaler "github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/gorilla/mux"
)

type Config struct {
	KubescalerConfig string
	KubeConfig       string
	ListenAddr       string
	UserDataFile     string
}

type API struct {
	ks  *kubescaler.Kubescaler
	srv http.Server
}

func New(conf Config) (*API, error) {
	log.Infof("setup kubescaler...")

	ks, err := kubescaler.New(conf.KubeConfig, conf.KubescalerConfig, conf.UserDataFile)
	if err != nil {
		return nil, errors.Wrap(err, "setup kubescaler")
	}
	handlerV1, err := v1.New(ks)
	if err != nil {
		return nil, errors.Wrap(err, "setup router")
	}
	h, err := handlers.RegisterRouter(handlerV1)
	if err != nil {
		return nil, errors.Wrap(err, "setup handlers")
	}

	return &API{
		ks: ks,
		srv: http.Server{
			Addr:         conf.ListenAddr,
			Handler:      h,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}, nil
}

func (a *API) Start(stopCh <-chan struct{}) error {
	a.ks.Run(stopCh)

	log.Infof("capacityservice: listen on %q", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *API) Mux() (m *mux.Router, err error) {
	m, ok := a.srv.Handler.(*mux.Router)
	if !ok {
		return nil, errors.New("Invalid type. Are you sure the API struct is initialized?")
	}
	return
}

func (a *API) Shutdown() error {
	return nil
}
