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

func (h Handler) AddUpdate(key string, pod v1.Pod) error {
	h.logger.Info(fmt.Sprintf("received pod add/update %s, ns %s name %s phase %s", key, pod.Namespace, pod.Name, pod.Status.Phase))
	return nil
}

func (h Handler) Delete(key string) error {
	h.logger.Info(fmt.Sprintf("received pod delete %s", key))
	return nil
}
