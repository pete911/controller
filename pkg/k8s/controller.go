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

const (
	reSyncInformer time.Duration = 0
	annotationKey                = "controller"
)

type podWorker interface {
	run(queue workqueue.RateLimitingInterface, indexer cache.KeyGetter)
}

type PodController struct {
	logger   *slog.Logger
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
	worker   podWorker
}

func NewPodController(logger *slog.Logger, clientset *kubernetes.Clientset, namespace string, handler PodHandler) (*PodController, error) {
	controller := &PodController{
		logger:   logger.With("component", "controller"),
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informer: newPodInformer(clientset, namespace, reSyncInformer),
		worker:   newWorker(logger, handler),
	}

	if _, err := controller.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addFunc,
		UpdateFunc: controller.updateFunc,
		DeleteFunc: controller.deleteFunc,
	}); err != nil {
		return nil, fmt.Errorf("add event handlers: %w", err)
	}
	return controller, nil
}

func newPodInformer(clientset *kubernetes.Clientset, namespace string, resyncPeriod time.Duration) cache.SharedIndexInformer {
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
		resyncPeriod,
		cache.Indexers{},
	)
}

func (c *PodController) addFunc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle add event: meta namespace key func: %v", err))
		return
	}
	if _, ok := toPod(obj).Annotations[annotationKey]; !ok {
		c.logger.Debug(fmt.Sprintf("add event %s does not have %s annotation, skipping", key, annotationKey))
		return
	}
	c.logger.Debug(fmt.Sprintf("add event %s added to queue", key))
	c.queue.Add(key)
}

func (c *PodController) updateFunc(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle update event: meta namespace key func: %v", err))
		return
	}
	if _, ok := toPod(newObj).Annotations[annotationKey]; !ok {
		c.logger.Debug(fmt.Sprintf("update event %s does not have %s annotation, skipping", key, annotationKey))
		return
	}
	c.logger.Debug(fmt.Sprintf("update event %s added to queue", key))
	c.queue.Add(key)
}

func (c *PodController) deleteFunc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle delete event: meta namespace key func: %v", err))
		return
	}
	if _, ok := toPod(obj).Annotations[annotationKey]; !ok {
		c.logger.Debug(fmt.Sprintf("delete event %s does not have %s annotation, skipping", key, annotationKey))
		return
	}
	c.logger.Debug(fmt.Sprintf("delete event %s added to queue", key))
	c.queue.Add(key)
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
	c.worker.run(c.queue, c.informer.GetIndexer())
	c.logger.Info("controller worker stopped")
}

func toPod(obj interface{}) v1.Pod {
	if obj == nil {
		return v1.Pod{}
	}
	return *obj.(*v1.Pod)
}
