package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmassimi/keda-go/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := config.Load()
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
	)
	if err != nil {
		log.Fatalf("create kafka client: %v", err)
	}
	defer client.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("receiver started brokers=%v topic=%s group=%s", cfg.Brokers, cfg.Topic, cfg.ConsumerGroup)
	for {
		fetches := client.PollFetches(ctx)
		if ctx.Err() != nil {
			log.Println("receiver stopped")
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Printf("fetch error topic=%s partition=%d: %v", e.Topic, e.Partition, e.Err)
			}
			continue
		}
		fetches.EachRecord(func(r *kgo.Record) {
			log.Printf("received topic=%s partition=%d offset=%d value=%q", r.Topic, r.Partition, r.Offset, string(r.Value))
			if cfg.ProcessDelay > 0 {
				select {
				case <-ctx.Done():
				case <-time.After(cfg.ProcessDelay):
				}
			}
		})
	}
}
