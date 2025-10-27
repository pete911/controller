package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Pod struct {
	logger    *slog.Logger
	clientset *kubernetes.Clientset
}

func NewPod(logger *slog.Logger, clientset *kubernetes.Clientset) *Pod {
	h := &Pod{
		logger:    logger.With("component", "handler"),
		clientset: clientset,
	}
	return h
}

func (h *Pod) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				return h.clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
			},
			WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				return h.clientset.CoreV1().Pods("").Watch(context.Background(), metav1.ListOptions{})
			},
		},
		&v1.Pod{},
		0,
		cache.Indexers{},
	)
}

func (h *Pod) AddOrUpdate(key string, value interface{}) error {
	pod := h.valueToPod(value)
	if pod.Status.PodIP == "" {
		h.logger.Debug(fmt.Sprintf("pod %s in phase %s does not have IP, skipping", key, pod.Status.Phase))
		return nil
	}

	h.logger.Info(fmt.Sprintf("processing pod event %s IP %s", key, pod.Status.PodIP))
	time.Sleep(5 * time.Second) // pretend that we are doing some work on pod add/update
	h.logger.Info(fmt.Sprintf("processed pod event %s IP %s", key, pod.Status.PodIP))
	return nil
}

func (h *Pod) Delete(key string) error {
	h.logger.Info(fmt.Sprintf("processing delete pod event %s", key))
	time.Sleep(5 * time.Second) // pretend that we are doing some work on pod delete
	h.logger.Info(fmt.Sprintf("processed delete pod event %s", key))
	return nil
}

func (h *Pod) valueToPod(value interface{}) v1.Pod {
	if value == nil {
		return v1.Pod{}
	}
	return *value.(*v1.Pod)
}
