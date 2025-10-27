package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pete911/controller/pkg"
	"github.com/pete911/controller/pkg/controller"
	"github.com/pete911/controller/pkg/handler"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	flags := pkg.ParseFlags()
	logger := pkg.NewLogger(flags.SlogLevel())
	logger.Info(fmt.Sprintf("starting controller with flags: %+v", flags))

	if err := run(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("rest in cluster config: %v", err)
	}
	if err := startPodController(logger, restConfig); err != nil {
		logger.Error(err.Error())
	}
	return nil
}

func startPodController(logger *slog.Logger, cfg *rest.Config) error {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("clientset for config: %v", err)
	}

	h := handler.NewPod(logger, client)
	ctrl, err := controller.NewController(logger, h)
	if err != nil {
		return fmt.Errorf("new pod controller: %v", err)
	}

	ctrl.Run(getStopCh(logger))
	return fmt.Errorf("pod controller stopped")
}

// example of controller for custom objects (CRDs)
func startEndpointSvcController(logger *slog.Logger, cfg *rest.Config) error {
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("clientset for config: %v", err)
	}

	h := handler.NewEndpointSvc(logger, client)
	ctrl, err := controller.NewController(logger, h)
	if err != nil {
		return fmt.Errorf("new endpoint svc controller: %v", err)
	}

	ctrl.Run(getStopCh(logger))
	return fmt.Errorf("endpoint svc controller stopped")
}

func getStopCh(logger *slog.Logger) <-chan struct{} {
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		logger.Debug("listening for SIGINT and SIGTERM")
		s := <-sigCh
		logger.Info(fmt.Sprintf("received %s signal, stopping", s))
		close(stopCh)
	}()
	return stopCh
}
