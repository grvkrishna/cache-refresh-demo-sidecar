// cache-refresh-demo-sidecar is a minimal demo of the cache refresh flow:
//
//	POST /sidecar/cache/refresh {"id":"abc"}
//	  -> handler.go  sends the message to the Kafka topic
//	  -> kafka.go    consumer (in every pod) receives the message
//	  -> refresh.go  calls the repo refresh API, up to 3 attempts
//
// main.go only reads settings and connects the pieces together.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RefreshMessage is the request body, the Kafka message and the repo API body.
type RefreshMessage struct {
	ID string `json:"id"`
}

func main() {
	// ---- 1. Settings from environment variables (with defaults for local runs) ----
	brokers := env("KAFKA_BROKERS", "localhost:9092")
	topic := env("KAFKA_TOPIC", "cache-refresh")
	refreshURL := env("REFRESH_URL", "http://localhost:8080/cache/refresh")
	httpAddr := env("HTTP_ADDR", ":8090")
	hostname, _ := os.Hostname()
	podName := env("POD_NAME", hostname)

	// ctx is cancelled when Kubernetes stops the pod (SIGTERM) or on Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---- 2. Create the pieces (NewXxx functions play the role of constructors) ----
	producer, err := NewProducer(brokers, topic)
	if err != nil {
		log.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	refresher := NewRefresher(refreshURL)

	// Each pod gets its own consumer group, so EVERY pod receives EVERY message.
	consumer, err := NewConsumer(brokers, topic, "cache-refresh-demo-"+podName, refresher)
	if err != nil {
		log.Fatalf("create consumer: %v", err)
	}
	defer consumer.Close()

	handler := NewHandler(producer)

	// ---- 3. Start the consumer in the background ("go" runs it concurrently) ----
	go consumer.Run(ctx)

	// ---- 4. Start the HTTP server in the background ----
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sidecar/cache/refresh", handler.Refresh)
	server := &http.Server{Addr: httpAddr, Handler: mux}

	go func() {
		log.Printf("[main] pod=%s listening on %s, topic=%s, repo=%s", podName, httpAddr, topic, refreshURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	// ---- 5. Wait for shutdown, then stop cleanly ----
	<-ctx.Done()
	log.Println("[main] shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx) // finish in-progress requests first
	// deferred consumer.Close() and producer.Close() run after this
}

func env(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
