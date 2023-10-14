package k8s

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log/slog"
	"os"
	"time"
)

type LeaderElection struct {
	logger    *slog.Logger
	client    *kubernetes.Clientset
	namespace string
}

func NewLeaderElection(logger *slog.Logger, clientset *kubernetes.Clientset, namespace string) LeaderElection {
	return LeaderElection{
		logger:    logger.With("component", "leaderelection"),
		client:    clientset,
		namespace: namespace,
	}
}

func (le LeaderElection) Run(ctx context.Context, fn func(ctx context.Context)) error {
	leaderElectionConfig := le.getConfig(fn)
	leaderElector, err := leaderelection.NewLeaderElector(leaderElectionConfig)
	if err != nil {
		return err
	}
	if leaderElectionConfig.WatchDog != nil {
		leaderElectionConfig.WatchDog.SetLeaderElection(leaderElector)
	}
	leaderElector.Run(ctx)
	// TODO test if we need to log etc.
	return nil
}

func (le LeaderElection) getConfig(fn func(ctx context.Context)) leaderelection.LeaderElectionConfig {
	return leaderelection.LeaderElectionConfig{
		Lock:            le.getLeaseLock("controller-lock"),
		ReleaseOnCancel: true,
		LeaseDuration:   30 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				le.logger.Info("started leading")
				fn(ctx)
			},
			OnStoppedLeading: func() {
				le.logger.Info("stopped leading")
				// TODO test if we need to exit and it doesn't keep restarting pods
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				le.logger.Info(fmt.Sprintf("new leader %s", identity))
			},
		},
	}
}

func (le LeaderElection) getLeaseLock(name string) *resourcelock.LeaseLock {
	return &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: le.namespace,
		},
		Client: le.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: uuid.New().String(),
		},
	}
}
