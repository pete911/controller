package main

import (
	"context"
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
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("rest in cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("clientset for config: %v", err)
	}

	namespace, err := getNamespace()
	if err != nil {
		return err
	}

	podHandler := handler.NewHandler(logger)
	podController, err := k8s.NewPodController(logger, clientset, podHandler)
	if err != nil {
		return fmt.Errorf("new pod informer: %v", err)
	}

	leaderElection := k8s.NewLeaderElection(logger, clientset, namespace)
	if err := leaderElection.Run(context.Background(), func(ctx context.Context) { podController.Run(ctx.Done()) }); err != nil {
		return fmt.Errorf("leader election run: %v", err)
	}
	return nil
}

func getNamespace() (string, error) {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("get namespace: %w", err)
	}
	return string(ns), nil
}
