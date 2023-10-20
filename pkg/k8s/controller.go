package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log/slog"
	"time"
)

const reSyncInformer time.Duration = 0

type PodController struct {
	logger   *slog.Logger
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
	worker   *podWorker
}

func NewPodController(logger *slog.Logger, clientset *kubernetes.Clientset, namespace string, handler PodHandler) (*PodController, error) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer := newPodInformer(clientset, namespace)
	processor := newPodWorker(logger, queue, informer.GetIndexer(), handler)

	controller := &PodController{
		logger:   logger.With("component", "controller"),
		queue:    queue,
		informer: informer,
		worker:   processor,
	}
	if err := controller.addEventHandlers(); err != nil {
		return nil, fmt.Errorf("add event handlers: %w", err)
	}
	return controller, nil
}

func newPodInformer(clientset *kubernetes.Clientset, namespace string) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{})
			},
		},
		&v1.Pod{},
		reSyncInformer,
		cache.Indexers{},
	)
}

func (c *PodController) addEventHandlers() error {
	_, err := c.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				c.logger.Error(fmt.Sprintf("handle add event: meta namespace key func: %v", err))
				return
			}
			c.logger.Debug(fmt.Sprintf("add event %s added to queue", key))
			c.queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				c.logger.Error(fmt.Sprintf("handle update event: meta namespace key func: %v", err))
				return
			}
			c.logger.Debug(fmt.Sprintf("update event %s added to queue", key))
			c.queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				c.logger.Error(fmt.Sprintf("handle delete event: meta namespace key func: %v", err))
				return
			}
			c.logger.Debug(fmt.Sprintf("delete event %s added to queue", key))
			c.queue.Add(key)
		},
	})
	return err
}

func (c *PodController) Run(stopCh <-chan struct{}) {
	c.logger.Info("starting controller")
	go func() {
		c.informer.Run(stopCh)
		c.logger.Info("informer stopped")
		c.queue.ShutDown()
		c.logger.Info("queue shut down")
	}()

	c.logger.Info("waiting for cache sync")
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		c.logger.Error("failed to sync")
		return
	}
	c.logger.Info("cache synced")
	c.logger.Info("starting controller worker")
	c.worker.run()
	c.logger.Info("controller worker stopped")
}
