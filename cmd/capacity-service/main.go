package main

import (
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"

	"github.com/supergiant/capacity/pkg/capacityserver"
	"github.com/supergiant/capacity/pkg/log"
)

type args struct {
	KubescalerConfig string `arg:"--kubescaler-config" env:"CAPACITY_KUBESCALER_CONFIG" help:"path to a kubescaler config"`
	KubeConfig       string `arg:"--kube-config"       env:"CAPACITY_KUBE_CONFIG"       help:"path to a kubeconfig file"`
	ListenAddr       string `arg:"--listen-addr"       env:"CAPACITY_LISTEN_ADDR"       help:"address to listen on, pass as a addr:port"`
	LogLevel         string `arg:"--verbosity"         env:"CAPACITY_LOG_LEVEL"         help:"logging verbosity"`
	LogHooks         string `arg:"--log-hooks"         env:"CAPACITY_LOG_HOOKS"         help:"list of comma-separated log providers (syslog)"`
	UserDataFile     string `arg:"--user-data"         env:"CAPACITY_USER_DATA"         help:"path to a userdata file"`
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
	log.SetOutput(os.Stdout)
	log.SetLevel(args.LogLevel)
	for _, hook := range strings.Split(args.LogHooks, ",") {
		if err := log.AddHook(hook); err != nil {
			log.Errorf("capacityserver: logger: add %s hook: %v", err)
		}
	}

	srv, err := capacityserver.New(capacityserver.Config{
		KubescalerConfig: args.KubescalerConfig,
		KubeConfig:       args.KubeConfig,
		ListenAddr:       args.ListenAddr,
		UserDataFile:     args.UserDataFile,
	})
	if err != nil {
		log.Fatalf("capacityserver: %v\n", err)
	}

	stopCh := signals.SetupSignalHandler()
	if err = srv.Start(stopCh); err != nil {
		log.Fatalf("capacityserver: start: %v\n", err)
	}
}
