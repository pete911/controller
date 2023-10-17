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
	stopCh := make(chan struct{})
	defer close(stopCh)

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

	podController.Run(stopCh)
	return nil
}
