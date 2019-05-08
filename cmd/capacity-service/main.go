package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/gobuffalo/packr"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/supergiant/capacity/pkg/capacityserver"
	"github.com/supergiant/capacity/pkg/kubescaler"
	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/version"
)

type args struct {
	KubescalerConfig   string `arg:"--kubescaler-config,   env:CAPACITY_KUBESCALER_CONFIG"   help:"path to a kubescaler config"`
	ConfigMapName      string `arg:"--configmap-name,      env:CAPACITY_CONFIGMAP_NAME"      help:"name of configMap with the 'kubescaler.conf' file"`
	ConfigMapNamespace string `arg:"--configmap-namespace, env:CAPACITY_CONFIGMAP_NAMESPACE" help:"namespace of configMap with kubescaler config"`
	KubeConfig         string `arg:"--kubeconfig,          env:CAPACITY_KUBE_CONFIG"         help:"path to a kubeconfig file, needs for building a kubernetes client"`
	ListenAddr         string `arg:"--listen-addr,         env:CAPACITY_LISTEN_ADDR"         help:"address to listen on, pass as a addr:port"`
	LogLevel           string `arg:"--log-level,           env:CAPACITY_LOG_LEVEL"           help:"logging verbosity [debug info warn error fatal panic]"`
	LogFormat          string `arg:"--log-format,          env:CAPACITY_LOG_LEVEL"           help:"logging format [txt json]"`
	LogHooks           string `arg:"--log-hooks,           env:CAPACITY_LOG_HOOKS"           help:"list of comma-separated log providers (syslog)"`
}

func (args) Version() string {
	return fmt.Sprintf("Version: %#v\n", version.Get())
}

func main() {
	args := args{
		ListenAddr: ":8081",
		LogLevel:   "info",
		LogFormat:  "txt",
	}
	arg.MustParse(&args)

	// setup logger
	configureLogging(args.LogLevel, args.LogFormat)

	for _, hook := range strings.Split(args.LogHooks, ",") {
		if err := log.AddHook(hook); err != nil {
			log.Errorf("capacityserver: logger: add %s hook: %v", hook, err)
		}
	}

	srv, err := capacityserver.New(capacityserver.Config{
		KubescalerOptions: kubescaler.Options{
			ConfigFile:         args.KubescalerConfig,
			ConfigMapName:      args.ConfigMapName,
			ConfigMapNamespace: args.ConfigMapNamespace,
			Kubeconfig:         args.KubeConfig,
		},
		ListenAddr: args.ListenAddr,
	})
	if err != nil {
		log.Fatalf("capacityserver: %v\n", err)
	}

	// register UI static file server
	mux, err := srv.Mux()
	if err != nil {
		log.Fatalf("Could not attach UI server to mux: %v\n", err)
	}
	uiFiles := packr.NewBox("./ui/capacity-service/dist")
	mux.PathPrefix("/ui/").Handler(
		http.StripPrefix("/ui/", http.FileServer(uiFiles)),
	)
	mux.Handle("/ui", http.RedirectHandler("../ui/", http.StatusMovedPermanently))
	mux.Handle("/", http.RedirectHandler("./ui", http.StatusMovedPermanently))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		<-signals.SetupSignalHandler()
	}()
	if err = srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func configureLogging(level, format string) {
	log.SetLevel(level)

	switch strings.TrimSpace(format) {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}
