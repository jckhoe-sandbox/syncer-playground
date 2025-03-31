package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/config"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	chat.UnimplementedChatServiceServer
	db            *gorm.DB
	eventManager  *events.RedisEventManager
	eventChannels []chan *chat.DataChangeEvent
	mu            sync.RWMutex
}

func (s *server) StreamDataChanges(req *chat.StreamDataChangesRequest, stream chat.ChatService_StreamDataChangesServer) error {
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

func (s *server) startRedisSubscriber(ctx context.Context) error {
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

func (s *server) startPostgresReplicator(ctx context.Context) error {
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

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.GetPostgresDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	eventManager, err := events.NewRedisEventManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create event manager: %v", err)
	}
	defer eventManager.Close()

	srv := &server{
		db:            db,
		eventManager:  eventManager,
		eventChannels: make([]chan *chat.DataChangeEvent, 0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.startRedisSubscriber(ctx); err != nil {
		log.Fatalf("Failed to start Redis subscriber: %v", err)
	}

	if err := srv.startPostgresReplicator(ctx); err != nil {
		log.Fatalf("Failed to start PostgreSQL replicator: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, srv)
	reflection.Register(s)

	log.Printf("Server listening on port %d", cfg.Server.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
