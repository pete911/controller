package controller

import (
	"fmt"
	"log/slog"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type worker interface {
	run(queue workqueue.TypedRateLimitingInterface[any], indexer cache.KeyGetter)
}

type Controller struct {
	logger   *slog.Logger
	queue    workqueue.TypedRateLimitingInterface[any]
	informer cache.SharedIndexInformer
	worker   worker
}

func NewController(logger *slog.Logger, handler Handler) (*Controller, error) {
	controller := &Controller{
		logger:   logger.With("component", "controller"),
		queue:    workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[any]()),
		informer: handler.Informer(),
		worker:   newQueueWorker(logger, handler),
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

func (c *Controller) addFunc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle add event: meta namespace key func: %v", err))
		return
	}
	c.logger.Debug(fmt.Sprintf("add event %s added to queue", key))
	c.queue.Add(key)
}

func (c *Controller) updateFunc(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle update event: meta namespace key func: %v", err))
		return
	}
	c.logger.Debug(fmt.Sprintf("update event %s added to queue", key))
	c.queue.Add(key)
}

func (c *Controller) deleteFunc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error(fmt.Sprintf("handle delete event: meta namespace key func: %v", err))
		return
	}
	c.logger.Debug(fmt.Sprintf("delete event %s added to queue", key))
	c.queue.Add(key)
}

func (c *Controller) Run(stopCh <-chan struct{}) {
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
