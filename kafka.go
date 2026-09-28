package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

// ---------------------------------------------------------------------------
// Producer: sends messages to the topic.
// ---------------------------------------------------------------------------

type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(brokers, topic string) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(strings.Split(brokers, ",")...),
	)
	if err != nil {
		return nil, err
	}
	return &Producer{client: client, topic: topic}, nil
}

// Send waits until Kafka confirms the message is stored.
func (p *Producer) Send(ctx context.Context, msg RefreshMessage) error {
	value, err := json.Marshal(msg) // {"id":"abc"}
	if err != nil {
		return err
	}
	record := &kgo.Record{Topic: p.topic, Value: value}
	return p.client.ProduceSync(ctx, record).FirstErr()
}

func (p *Producer) Close() {
	p.client.Close()
}

// ---------------------------------------------------------------------------
// Consumer: reads messages from the topic and triggers the refresh.
// ---------------------------------------------------------------------------

type Consumer struct {
	client    *kgo.Client
	refresher *Refresher
}

func NewConsumer(brokers, topic, groupID string, refresher *Refresher) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(strings.Split(brokers, ",")...),
		kgo.ConsumerGroup(groupID), // unique per pod -> every pod gets every message
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()), // a new pod skips old messages
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{client: client, refresher: refresher}, nil
}

// Run keeps reading messages until ctx is cancelled (shutdown).
func (c *Consumer) Run(ctx context.Context) {
	log.Println("[consumer] started")
	for {
		fetches := c.client.PollFetches(ctx) // waits for new messages
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			log.Printf("[consumer] fetch error topic=%s partition=%d: %v", topic, partition, err)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			var msg RefreshMessage
			if err := json.Unmarshal(record.Value, &msg); err != nil {
				log.Printf("[consumer] skipping invalid message %q: %v", record.Value, err)
				return
			}
			log.Printf("[consumer] received id=%s", msg.ID)
			c.refresher.Refresh(msg)
		})
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}
