package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmassimi/keda-go/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := config.Load()
	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...), kgo.DefaultProduceTopic(cfg.Topic))
	if err != nil {
		log.Fatalf("create kafka client: %v", err)
	}
	defer client.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			body = []byte(fmt.Sprintf("message-%d", time.Now().UnixNano()))
		}
		rec := &kgo.Record{Topic: cfg.Topic, Value: body}
		if err := client.ProduceSync(r.Context(), rec).FirstErr(); err != nil {
			log.Printf("produce failed: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		log.Printf("sent to topic=%s value=%q", cfg.Topic, string(body))
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("sent\n"))
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
	go func() {
		log.Printf("sender listening on %s, brokers=%v topic=%s", cfg.HTTPAddr, cfg.Brokers, cfg.Topic)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
