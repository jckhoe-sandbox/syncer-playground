package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/protobuf/types/known/timestamppb"

	"syncer-playground/pkg/chat"
	"syncer-playground/pkg/config"
)

const (
	dataChangeChannel = "data_changes"
)

type RedisEventManager struct {
	client *redis.Client
}

func NewRedisEventManager(cfg *config.Config) (*RedisEventManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisEventManager{
		client: client,
	}, nil
}

// PublishEvent publishes a data change event to Redis
func (m *RedisEventManager) PublishEvent(ctx context.Context, event *chat.DataChangeEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := m.client.Publish(ctx, dataChangeChannel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// SubscribeToEvents subscribes to data change events from Redis
func (m *RedisEventManager) SubscribeToEvents(ctx context.Context) (<-chan *chat.DataChangeEvent, error) {
	pubsub := m.client.Subscribe(ctx, dataChangeChannel)
	eventChan := make(chan *chat.DataChangeEvent, 100)

	// Start subscription in a goroutine
	go func() {
		defer pubsub.Close()
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				var event chat.DataChangeEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Printf("Error unmarshaling event: %v", err)
					continue
				}

				// Set timestamp if not set
				if event.Timestamp == nil {
					event.Timestamp = timestamppb.Now()
				}

				eventChan <- &event
			}
		}
	}()

	return eventChan, nil
}

func (m *RedisEventManager) Close() error {
	return m.client.Close()
} 