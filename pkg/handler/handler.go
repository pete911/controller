package handler

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"log/slog"
	"sync"
	"time"
)

const podChannelBuff = 10

type PodEvent struct {
	Key string
	IP  string
}

func NewPodEvent(key string, pod v1.Pod) PodEvent {
	return PodEvent{Key: key, IP: pod.Status.PodIP}
}

type Handler struct {
	logger    *slog.Logger
	mux       *sync.RWMutex
	podEvents map[string]chan<- PodEvent
}

func NewHandler(logger *slog.Logger) *Handler {
	h := &Handler{
		logger:    logger.With("component", "handler"),
		mux:       &sync.RWMutex{},
		podEvents: make(map[string]chan<- PodEvent),
	}
	return h
}

func (h *Handler) AddUpdate(key string, pod v1.Pod) error {
	if pod.Status.PodIP == "" {
		h.logger.Debug(fmt.Sprintf("pod %s in phase %s does not have IP, skipping", key, pod.Status.Phase))
		return nil
	}

	h.logger.Info(fmt.Sprintf("received pod add/update %s phase %s IP %s", key, pod.Status.Phase, pod.Status.PodIP))
	h.getPodChannel(key) <- NewPodEvent(key, pod)
	return nil
}

func (h *Handler) Delete(key string) error {
	h.logger.Info(fmt.Sprintf("received pod delete %s, deleting channel", key))
	h.deletePodChannel(key)
	return nil
}

func (h *Handler) getPodChannel(key string) chan<- PodEvent {
	h.mux.Lock()
	defer h.mux.Unlock()
	if c, ok := h.podEvents[key]; ok {
		h.logger.Debug(fmt.Sprintf("using existing channel for %s", key))
		return c
	}
	h.podEvents[key] = h.newPodChannel(key)
	h.logger.Debug(fmt.Sprintf("created new channel for %s", key))
	return h.podEvents[key]
}

func (h *Handler) deletePodChannel(key string) {
	h.mux.Lock()
	defer h.mux.Unlock()
	if c, ok := h.podEvents[key]; ok {
		close(c)
		h.logger.Debug(fmt.Sprintf("%s channel closed", key))
		delete(h.podEvents, key)
		h.logger.Debug(fmt.Sprintf("key %s deleted from pod events map", key))
		return
	}
	h.logger.Info(fmt.Sprintf("channel for %s not found", key))
}

// newPodChannel creates new channel for a specific pod and return it
func (h *Handler) newPodChannel(key string) chan<- PodEvent {
	events := make(chan PodEvent, podChannelBuff)
	go func() {
		for e := range events {
			h.logger.Info(fmt.Sprintf("got pod event %s IP %s", e.Key, e.IP))
			time.Sleep(1 * time.Second) // pretend that we are doing some work on pod add/update
		}
		// channel closed, pod has been deleted
		time.Sleep(1 * time.Second) // pretend that we are doing some work on pod delete
		h.logger.Debug(fmt.Sprintf("finished processing %s events", key))
	}()
	return events
}
