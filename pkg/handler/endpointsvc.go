package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/pete911/controller/pkg/types"

	ackec2apis "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

type EndpointSvc struct {
	logger *slog.Logger
	client dynamic.Interface
}

func NewEndpointSvc(logger *slog.Logger, client dynamic.Interface) *EndpointSvc {
	h := &EndpointSvc{
		logger: logger.With("component", "handler", "name", "endpoint svc"),
		client: client,
	}
	return h
}

func (h *EndpointSvc) Informer() cache.SharedIndexInformer {
	gvr := schema.GroupVersionResource{Group: "ec2.services.k8s.aws", Version: "v1alpha1", Resource: "vpcendpointserviceconfigurations"}
	return dynamicinformer.NewDynamicSharedInformerFactory(h.client, 0).ForResource(gvr).Informer()
}

func (h *EndpointSvc) AddOrUpdate(key string, value interface{}) error {
	h.logger.Info(fmt.Sprintf("add or update endpoint service %s: received event", key))
	endpointSvc, err := h.valueToEndpointService(value)
	if err != nil {
		return err
	}

	if endpointSvc.Status.PrivateDNSNameConfiguration == nil {
		h.logger.Debug(fmt.Sprintf("endpoint svc %s does not have private dns name configuraion set, skipping", key))
		return nil
	}
	dnsNameConfiguration := types.ToDnsNameConfiguration(endpointSvc.Status.PrivateDNSNameConfiguration)

	// TODO insert your code to do whatever

	h.logger.Info(fmt.Sprintf("endpoint service %s: private dns name configuraion: %+v", key, dnsNameConfiguration))

	time.Sleep(5 * time.Second) // pretend that we are doing some work on pod add/update
	h.logger.Info(fmt.Sprintf("add or update endpoint service %s: processed event", key))
	return nil
}

func (h *EndpointSvc) Delete(key string) error {
	h.logger.Info(fmt.Sprintf("delete vpc endpoint service %s: received event", key))
	time.Sleep(5 * time.Second) // TODO pretend that we are doing some work on delete

	// TODO insert your code to do whatever

	// delete resource
	h.logger.Info(fmt.Sprintf("delete vpc endpoint service %s: processed event", key))
	return nil
}

func (h *EndpointSvc) valueToEndpointService(value interface{}) (ackec2apis.VPCEndpointServiceConfiguration, error) {
	if value == nil {
		return ackec2apis.VPCEndpointServiceConfiguration{}, errors.New("cannot convert nil to VPCEndpointServiceConfiguration")
	}
	obj, ok := value.(*unstructured.Unstructured)
	if !ok {
		return ackec2apis.VPCEndpointServiceConfiguration{}, fmt.Errorf("object is %T type, expected unstructured", value)
	}
	var endpointSvc ackec2apis.VPCEndpointServiceConfiguration
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &endpointSvc); err != nil {
		return ackec2apis.VPCEndpointServiceConfiguration{}, fmt.Errorf(fmt.Sprintf("convert unstructured to VPCEndpointServiceConfiguration: %v", err))
	}
	return endpointSvc, nil
}
