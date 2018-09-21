package capacityserver

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/capacityserver/handlers"
	kubescaler "github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
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

	h, err := handlers.Handler(ks)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		a.ks.Run(stopCh)
	}()
	go a.srv.ListenAndServe()
	log.Infof("capacityservice: listen on %q", a.srv.Addr)

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down http server...")
			return a.srv.Shutdown(ctx)
		}
	}
}
