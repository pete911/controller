package handler

import (
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

func (h Handler) Add(pod v1.Pod) {
	h.logger.Info("received pod add")
}

func (h Handler) Update(oldPod, newPod v1.Pod) {
	h.logger.Info("received pod update")
}

func (h Handler) Delete(pod v1.Pod) {
	h.logger.Info("received pod delete")
}
