package handler

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"log/slog"
)

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) Handler {
	return Handler{
		logger: logger.With("component", "handler"),
	}
}

func (h Handler) Add(pod v1.Pod) error {
	h.logger.Info(fmt.Sprintf("received pod add, ns %s name %s phase %s", pod.Namespace, pod.Name, pod.Status.Phase))
	return nil
}

func (h Handler) Update(oldPod, newPod v1.Pod) error {
	h.logger.Info(fmt.Sprintf("received pod update, old ns %s name %s phase %s", oldPod.Namespace, oldPod.Name, oldPod.Status.Phase))
	h.logger.Info(fmt.Sprintf("received pod update, new ns %s name %s phase %s", newPod.Namespace, newPod.Name, newPod.Status.Phase))
	return nil
}

func (h Handler) Delete(pod v1.Pod) error {
	h.logger.Info(fmt.Sprintf("received pod delete, ns %s name %s phase %s", pod.Namespace, pod.Name, pod.Status.Phase))
	return nil
}
