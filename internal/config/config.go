package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
	HTTPAddr      string
	ProcessDelay  time.Duration
}

func Load() Config {
	return Config{
		Brokers:       split(getenv("KAFKA_BROKERS", "kafka.default.svc.cluster.local:9092")),
		Topic:         getenv("KAFKA_TOPIC", "injectMessage"),
		ConsumerGroup: getenv("KAFKA_CONSUMER_GROUP", "go-receiver-group"),
		HTTPAddr:      getenv("HTTP_ADDR", ":9999"),
		ProcessDelay:  time.Duration(getenvInt("PROCESS_DELAY_MS", 250)) * time.Millisecond,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func split(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
