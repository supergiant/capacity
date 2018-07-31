package main

import (
	"github.com/alexflint/go-arg"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/supergiant/capacity/pkg/capacityserver"
	"github.com/supergiant/capacity/pkg/log"
	"os"
)

type args struct {
	KubescalerConfig string `arg:"--kubescaler-config" env:"CAPACITY_KUBESCALER_CONFIG" help:"path to a kubescaler config"`
	KubeConfig       string `arg:"--kube-config"       env:"CAPACITY_KUBE_CONFIG"       help:"path to a kubeconfig file"`
	ListenAddr       string `arg:"--listen-addr"       env:"CAPACITY_LISTEN_ADDR"       help:"address to listen on, pass as a addr:port"`
	LogLevel         string `arg:"--verbosity"         env:"CAPACITY_LOG_LEVEL"         help:"logging verbosity"`
}

func (args) Version() string {
	return "raw"
}

func main() {
	args := &args{
		KubescalerConfig: "/etc/kubescaler.conf",
		ListenAddr:       ":8081",
		LogLevel:         "info",
	}
	arg.MustParse(args)

	// setup logger
	log.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(args.LogLevel)

	srv, err := capacityserver.New(capacityserver.Config{
		KubescalerConfig: args.KubescalerConfig,
		KubeConfig:       args.KubeConfig,
		ListenAddr:       args.ListenAddr,
	})
	if err != nil {
		log.Fatalf("capacityserver: %v\n", err)
	}

	stopCh := signals.SetupSignalHandler()
	if err = srv.Start(stopCh); err != nil {
		log.Fatalf("capacityserver: start: %v\n", err)
	}
}
