package main

import (
	"context"
	"fmt"
	"github.com/pete911/controller/pkg"
	"github.com/pete911/controller/pkg/handler"
	"github.com/pete911/controller/pkg/k8s"
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

	namespace, err := getNamespace()
	if err != nil {
		return err
	}

	podInformer, err := k8s.NewPodInformer(logger, restConfig, "")
	if err != nil {
		return fmt.Errorf("new pod informer: %v", err)
	}

	podHandler := handler.NewHandler(logger)
	if err := podInformer.AddEventHandlers(podHandler); err != nil {
		return fmt.Errorf("add pod informer event handlers: %v", err)
	}

	leaderElection, err := k8s.NewLeaderElection(logger, restConfig, namespace)
	if err != nil {
		return fmt.Errorf("new leader election: %v", err)
	}

	if err := leaderElection.Run(context.Background(), func(ctx context.Context) { podInformer.Run(ctx.Done()) }); err != nil {
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
