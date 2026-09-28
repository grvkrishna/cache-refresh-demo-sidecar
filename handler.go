package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// Handler serves the HTTP endpoint. It only needs the producer.
type Handler struct {
	producer *Producer
}

func NewHandler(producer *Producer) *Handler {
	return &Handler{producer: producer}
}

// Refresh handles POST /sidecar/cache/refresh with body {"id":"abc"}.
// It does NOT refresh anything itself: it sends the message to Kafka, and every
// pod's consumer (including this one) does the refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var msg RefreshMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil || msg.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `body must be JSON like {"id":"abc"}`})
		return
	}

	if err := h.producer.Send(r.Context(), msg); err != nil {
		log.Printf("[api] could not send id=%s to kafka: %v", msg.ID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to send message to kafka"})
		return
	}

	log.Printf("[api] sent id=%s to kafka", msg.ID)
	writeJSON(w, http.StatusAccepted, map[string]string{"id": msg.ID, "status": "sent to kafka"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
