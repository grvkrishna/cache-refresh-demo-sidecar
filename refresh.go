package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	maxAttempts = 3
	retryWait   = 2 * time.Second
)

// Refresher calls the repo's refresh API.
type Refresher struct {
	url    string
	client *http.Client
}

func NewRefresher(url string) *Refresher {
	return &Refresher{
		url:    url,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Refresh tries up to maxAttempts times, waiting retryWait between attempts.
// If every attempt fails it only logs (no failure topic in the demo).
func (r *Refresher) Refresh(msg RefreshMessage) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.callRepo(msg)
		if err == nil {
			log.Printf("[refresh] SUCCESS id=%s (attempt %d)", msg.ID, attempt)
			return
		}

		log.Printf("[refresh] attempt %d/%d failed id=%s: %v", attempt, maxAttempts, msg.ID, err)
		if attempt < maxAttempts {
			time.Sleep(retryWait)
		}
	}
	log.Printf("[refresh] FAILED id=%s after %d attempts - manual retry needed", msg.ID, maxAttempts)
}

// callRepo makes one POST to the repo refresh API with the same {"id":"..."} body.
func (r *Refresher) callRepo(msg RefreshMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := r.client.Post(r.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err // repo not reachable, timeout, etc.
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("repo returned status %d", resp.StatusCode)
	}
	return nil
}
