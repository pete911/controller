package main

import (
	"fmt"
	"github.com/pete911/controller/pkg"
	"github.com/pete911/controller/pkg/handler"
	"github.com/pete911/controller/pkg/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("clientset for config: %v", err)
	}

	podHandler := handler.NewHandler(logger)
	podController, err := k8s.NewPodController(logger, clientset, "", podHandler)
	if err != nil {
		return fmt.Errorf("new pod informer: %v", err)
	}

	podController.Run(getStopCh(logger))
	logger.Info("controller stopped")
	return nil
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
