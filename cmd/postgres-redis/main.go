package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"syncer-playground/pkg/chat"
	"syncer-playground/pkg/config"
	"syncer-playground/pkg/events"
	"syncer-playground/pkg/replication"
)

type server struct {
	chat.UnimplementedChatServiceServer
	db            *gorm.DB
	replicator    *replication.PostgresReplicator
	eventManager  *events.RedisEventManager
	eventChannels []chan *chat.DataChangeEvent
	mu            sync.RWMutex
}

func (s *server) StreamDataChanges(req *chat.StreamDataChangesRequest, stream chat.ChatService_StreamDataChangesServer) error {
	// Create a channel for this client
	eventChan := make(chan *chat.DataChangeEvent, 100)

	// Register the channel
	s.mu.Lock()
	s.eventChannels = append(s.eventChannels, eventChan)
	s.mu.Unlock()

	// Cleanup when done
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

	// Send events to the client
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
	// Subscribe to Redis events
	eventChan, err := s.eventManager.SubscribeToEvents(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to Redis events: %w", err)
	}

	// Forward Redis events to all connected clients
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
	// Create a channel for PostgreSQL events
	pgEventChan := make(chan *chat.DataChangeEvent, 100)

	// Start replication
	if err := s.replicator.StartReplication(ctx, pgEventChan); err != nil {
		return fmt.Errorf("failed to start replication: %w", err)
	}

	// Forward PostgreSQL events to Redis
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
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.GetPostgresDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create replication manager
	replicator, err := replication.NewPostgresReplicator(cfg)
	if err != nil {
		log.Fatalf("Failed to create replicator: %v", err)
	}
	defer replicator.Close()

	// Setup replication
	if err := replicator.SetupReplication(context.Background()); err != nil {
		log.Fatalf("Failed to setup replication: %v", err)
	}

	// Create Redis event manager
	eventManager, err := events.NewRedisEventManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create event manager: %v", err)
	}
	defer eventManager.Close()

	// Create server instance
	srv := &server{
		db:            db,
		replicator:    replicator,
		eventManager:  eventManager,
		eventChannels: make([]chan *chat.DataChangeEvent, 0),
	}

	// Create context for background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Redis subscriber
	if err := srv.startRedisSubscriber(ctx); err != nil {
		log.Fatalf("Failed to start Redis subscriber: %v", err)
	}

	// Start PostgreSQL replicator
	if err := srv.startPostgresReplicator(ctx); err != nil {
		log.Fatalf("Failed to start PostgreSQL replicator: %v", err)
	}

	// Create gRPC server
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