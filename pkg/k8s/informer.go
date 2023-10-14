package k8s

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"log/slog"
	"os"
	"sync"
	"time"
)

type PodHandler interface {
	Add(pod v1.Pod) error
	Update(oldPod, newPod v1.Pod) error
	Delete(pod v1.Pod) error
}

type PodInformer struct {
	logger   *slog.Logger
	mux      *sync.RWMutex
	synced   bool
	informer cache.SharedIndexInformer
}

func NewPodInformer(logger *slog.Logger, restConfig *rest.Config, namespace string) (*PodInformer, error) {
	dc, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	reSync := 60 * time.Second
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, reSync, namespace, nil)
	resource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	informer := factory.ForResource(resource).Informer()

	return &PodInformer{
		logger:   logger.With("component", "informer"),
		mux:      &sync.RWMutex{},
		synced:   false,
		informer: informer,
	}, nil
}

func (i *PodInformer) AddEventHandlers(handler PodHandler) error {
	_, err := i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			i.handlePodEvent("add", obj, handler.Delete)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			i.handlePodsEvent("update", oldObj, newObj, handler.Update)
		},
		DeleteFunc: func(obj interface{}) {
			i.handlePodEvent("delete", obj, handler.Delete)
		},
	})
	return err
}

func (i *PodInformer) handlePodEvent(name string, obj interface{}, fn func(pod v1.Pod) error) {
	i.mux.RLock()
	defer i.mux.RUnlock()
	if !i.synced {
		i.logger.Debug(fmt.Sprintf("skip %s event, informer is not synced", name))
		return
	}
	i.logger.Debug(fmt.Sprintf("handle %s event", name))
	if pod, ok := i.toPod(obj); ok {
		if err := fn(pod); err != nil {
			i.logger.Error(fmt.Sprintf("%s event: %v", name, err))
		}
	}
}

func (i *PodInformer) handlePodsEvent(name string, oldObj, newObj interface{}, fn func(oldPod, newPod v1.Pod) error) {
	i.mux.RLock()
	defer i.mux.RUnlock()
	if !i.synced {
		i.logger.Debug(fmt.Sprintf("skip %s event, informer is not synced", name))
		return
	}
	i.logger.Debug("handle update event")
	if oldPod, newPod, ok := i.toPods(oldObj, newObj); ok {
		if err := fn(oldPod, newPod); err != nil {
			i.logger.Error(fmt.Sprintf("%s event: %v", name, err))
		}
	}
}

func (i *PodInformer) toPod(obj interface{}) (v1.Pod, bool) {
	var pod v1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).Object, &pod); err != nil {
		i.logger.Error(fmt.Sprintf("convert object to pod: %v", err))
		return v1.Pod{}, false
	}
	return pod, true
}

func (i *PodInformer) toPods(oldObj, newObj interface{}) (v1.Pod, v1.Pod, bool) {
	var oldPod, newPod v1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(oldObj.(*unstructured.Unstructured).Object, &oldPod); err != nil {
		i.logger.Error(fmt.Sprintf("convert old object to pod: %v", err))
		return v1.Pod{}, v1.Pod{}, false
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newObj.(*unstructured.Unstructured).Object, &newPod); err != nil {
		i.logger.Error(fmt.Sprintf("convert new object to pod: %v", err))
		return v1.Pod{}, v1.Pod{}, false
	}
	return oldPod, newPod, true
}

func (i *PodInformer) Run(stopCh <-chan struct{}) {
	go i.informer.Run(stopCh)
	i.logger.Info("waiting for cache sync")
	isSynced := cache.WaitForCacheSync(stopCh, i.informer.HasSynced)
	i.logger.Info("cache synced")
	i.mux.Lock()
	i.synced = isSynced
	i.mux.Unlock()
	if !isSynced {
		i.logger.Error("failed to sync")
		os.Exit(1)
	}
	<-stopCh
}
