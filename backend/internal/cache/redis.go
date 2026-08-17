package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis connects to Redis and verifies it with a ping. Podman/macOS
// port-forwards can be slow to warm up, so the probe is generous: up to 30s.
func NewRedis(addr, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	deadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}
