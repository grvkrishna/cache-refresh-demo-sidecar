// fakekafka runs an in-memory Kafka on localhost:9092 for the demo (no Docker needed).
//
//	go run ./tools/fakekafka
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/twmb/franz-go/pkg/kfake"
)

func main() {
	cluster, err := kfake.NewCluster(
		kfake.Ports(9092),
		kfake.SeedTopics(1, "cache-refresh"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Close()
	log.Printf("fake kafka listening on %v (topic: cache-refresh)", cluster.ListenAddrs())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
