package main

import (
	"context"
	"fmt"
	"github.com/pete911/controller/pkg"
	"github.com/pete911/controller/pkg/handler"
	"github.com/pete911/controller/pkg/k8s"
	"k8s.io/client-go/tools/clientcmd"
	"log/slog"
	"os"
)

func main() {
	flags := pkg.ParseFlags()
	logger := pkg.NewLogger(flags.SlogLevel(), flags.LogJson)
	logger.Info(fmt.Sprintf("starting controller with flags: %+v", flags))

	if err := run(logger, flags); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger, flags pkg.Flags) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", flags.Kubeconfig)
	if err != nil {
		return fmt.Errorf("rest config from flags: %v", err)
	}

	podInformer, err := k8s.NewPodInformer(logger, restConfig, flags.Namespace)
	if err != nil {
		return fmt.Errorf("new pod informer: %v", err)
	}

	podHandler := handler.NewHandler(logger)
	if err := podInformer.AddEventHandlers(podHandler); err != nil {
		return fmt.Errorf("add pod informer event handlers: %v", err)
	}

	leaderElection, err := k8s.NewLeaderElection(logger, restConfig, flags.Namespace)
	if err != nil {
		return fmt.Errorf("new leader election: %v", err)
	}

	if err := leaderElection.Run(context.Background(), func(ctx context.Context) {}); err != nil {
		return fmt.Errorf("leader election run: %v", err)
	}
	return nil
}
