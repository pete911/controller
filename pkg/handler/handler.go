package handler

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"log/slog"
	"time"
)

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	h := &Handler{
		logger: logger.With("component", "handler"),
	}
	return h
}

func (h *Handler) AddOrUpdate(key string, pod v1.Pod) error {
	if pod.Status.PodIP == "" {
		h.logger.Debug(fmt.Sprintf("pod %s in phase %s does not have IP, skipping", key, pod.Status.Phase))
		return nil
	}

	h.logger.Info(fmt.Sprintf("processing pod event %s IP %s", key, pod.Status.PodIP))
	time.Sleep(5 * time.Second) // pretend that we are doing some work on pod add/update
	h.logger.Info(fmt.Sprintf("processed pod event %s IP %s", key, pod.Status.PodIP))
	return nil
}

func (h *Handler) Delete(key string) error {
	h.logger.Info(fmt.Sprintf("processing delete pod event %s", key))
	time.Sleep(5 * time.Second) // pretend that we are doing some work on pod delete
	h.logger.Info(fmt.Sprintf("processed delete pod event %s", key))
	return nil
}
