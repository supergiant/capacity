package capacityserver

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/supergiant/capacity/pkg/capacityserver/handlers"
	"github.com/supergiant/capacity/pkg/capacityserver/handlers/v1"
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
)

var (
	shutdownTimeout = time.Second * 30
)

type Config struct {
	ListenAddr        string
	KubescalerOptions kubescaler.Options
}

type API struct {
	ks  *kubescaler.Kubescaler
	srv http.Server
}

func New(conf Config) (*API, error) {
	log.Infof("setup kubescaler...")

	ks, err := kubescaler.New(conf.KubescalerOptions)
	if err != nil {
		return nil, errors.Wrap(err, "setup kubescaler")
	}

	handlerV1, err := v1.New(ks)
	if err != nil {
		return nil, errors.Wrap(err, "setup router")
	}

	h, err := handlers.RegisterRouter(ks, handlerV1)
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

func (a *API) Start(ctx context.Context) error {
	routines := []struct {
		name     string
		run      func() error
		shutdown func(ctx context.Context) error
	}{
		{
			name:     "kubescaler",
			run:      a.ks.Run,
			shutdown: a.ks.Stop,
		},
		{
			name: "web server",
			run: func() error {
				log.Infof("listen on %q", a.srv.Addr)
				err := a.srv.ListenAndServe()
				if err != nil || err != http.ErrServerClosed {
					return err
				}
				return nil
			},
			shutdown: a.srv.Shutdown,
		},
	}

	ctx, cancel := context.WithCancel(ctx)
	errsCh := make(chan error, len(routines)*2)
	wg := sync.WaitGroup{}
	for _, r := range routines {
		wg.Add(1)

		routine := r
		go func() {
			defer wg.Done()
			defer cancel()

			errch := make(chan error)
			// "shutdown" routine: exit on a canceled context
			go func() {
				<-ctx.Done()

				log.Infof("terminating %s (force exit after %s)", routine.name, shutdownTimeout.String())
				ctx, _ := context.WithTimeout(context.Background(), shutdownTimeout)
				errch <- errors.Wrapf(routine.shutdown(ctx), "%s: shutdown", routine.name)
			}()
			// "run" routine: exit on failure
			go func() {
				log.Infof("starting %s", routine.name)
				errch <- errors.Wrapf(routine.run(), "%s: run", routine.name)
			}()

			if err := <-errch; err != nil {
				errsCh <- err
			} else {
				log.Infof("%s has been stopped", routine.name)
			}
		}()
	}
	// wait until all routines will be done
	wg.Wait()
	close(errsCh)

	var failed bool
	for err := range errsCh {
		if err != nil {
			failed = true
			log.Error(err)
		}
	}

	return toErr(failed)
}

func (a *API) Mux() (m *mux.Router, err error) {
	m, ok := a.srv.Handler.(*mux.Router)
	if !ok {
		return nil, errors.New("Invalid type. Are you sure the API struct is initialized?")
	}
	return
}

func toErr(fail bool) error {
	if fail {
		return errors.New("graceful shutdown has been failed")
	}
	return nil
}
