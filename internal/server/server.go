package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/events"
	"gorm.io/gorm"
)

type Server struct {
	chat.UnimplementedChatServiceServer
	db            *gorm.DB
	eventManager  *events.RedisEventManager
	eventChannels []chan *chat.DataChangeEvent
	mu            sync.RWMutex
}

func NewServer(
	db *gorm.DB,
	eventManager *events.RedisEventManager,
) *Server {
	return &Server{
		db:           db,
		eventManager: eventManager,
	}
}

func (s *Server) StreamDataChanges(req *chat.StreamDataChangesRequest, stream chat.ChatService_StreamDataChangesServer) error {
	eventChan := make(chan *chat.DataChangeEvent, 100)

	s.mu.Lock()
	s.eventChannels = append(s.eventChannels, eventChan)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		for i, ch := range s.eventChannels {
			if ch == eventChan {
				s.eventChannels = append(s.eventChannels[:i], s.eventChannels[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		close(eventChan)
	}()

	for {
		select {
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				return fmt.Errorf("failed to send event: %w", err)
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *Server) StartRedisSubscriber(ctx context.Context) error {
	eventChan, err := s.eventManager.SubscribeToEvents(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to Redis events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-eventChan:
				s.mu.RLock()
				for _, ch := range s.eventChannels {
					select {
					case ch <- event:
					default:
						log.Printf("Warning: client channel is full, dropping event")
					}
				}
				s.mu.RUnlock()
			}
		}
	}()

	return nil
}

func (s *Server) StartPostgresReplicator(ctx context.Context) error {
	pgEventChan := make(chan *chat.DataChangeEvent, 100)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-pgEventChan:
				if err := s.eventManager.PublishEvent(ctx, event); err != nil {
					log.Printf("Error publishing event to Redis: %v", err)
				}
			}
		}
	}()

	return nil
}
