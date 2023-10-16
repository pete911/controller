package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log/slog"
	"time"
)

type PodHandler interface {
	AddUpdate(key string, pod v1.Pod) error
	Delete(key string) error
}

type PodController struct {
	logger          *slog.Logger
	queue           workqueue.RateLimitingInterface
	informer        cache.SharedIndexInformer
	handler         PodHandler
	maxQueueRetries int
}

func NewPodController(logger *slog.Logger, clientset *kubernetes.Clientset, handler PodHandler) (*PodController, error) {
	controller := &PodController{
		logger:          logger.With("component", "controller"),
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informer:        newPodInformer(clientset),
		handler:         handler,
		maxQueueRetries: 3,
	}
	if err := controller.addEventHandlers(); err != nil {
		return nil, fmt.Errorf("add event handlers: %w", err)
	}
	return controller, nil
}

func newPodInformer(clientset *kubernetes.Clientset) cache.SharedIndexInformer {
	reSync := 60 * time.Second
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.CoreV1().Pods(v1.NamespaceAll).Watch(context.Background(), metav1.ListOptions{})
			},
		},
		&v1.Pod{},
		reSync,
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
			c.logger.Info(fmt.Sprintf("add event %s added to queue", key))
			c.queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				c.logger.Error(fmt.Sprintf("handle update event: meta namespace key func: %v", err))
				return
			}
			c.logger.Info(fmt.Sprintf("update event %s added to queue", key))
			c.queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				c.logger.Error(fmt.Sprintf("handle delete event: meta namespace key func: %v", err))
				return
			}
			c.logger.Info(fmt.Sprintf("delete event %s added to queue", key))
			c.queue.Add(key)
		},
	})
	return err
}

func (c *PodController) Run(stopCh <-chan struct{}) {
	defer c.queue.ShutDown()
	c.logger.Info("starting controller")
	go c.informer.Run(stopCh)

	c.logger.Info("waiting for cache sync")
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		c.logger.Error("failed to sync")
		return
	}
	c.logger.Info("cache synced")
	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *PodController) runWorker() {
	for c.processNextItem() {
	}
}

func (c *PodController) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	if err := c.processItem(key.(string)); err != nil {
		retries := c.queue.NumRequeues(key)
		if retries < c.maxQueueRetries {
			c.logger.Error(fmt.Sprintf("queue retries %d (max retries %d)", retries, c.maxQueueRetries))
			return true
		}
		c.queue.Forget(key)
		c.logger.Error(fmt.Sprintf("process %s: %v", key, err))
	}

	c.queue.Forget(key)
	return true
}

func (c *PodController) processItem(key string) error {
	var pod v1.Pod
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("get object by key %s from store: %w", key, err)
	}
	if !exists {
		return c.handler.Delete(key)
	}
	if obj != nil {
		pod = *obj.(*v1.Pod)
	}
	return c.handler.AddUpdate(key, pod)
}
