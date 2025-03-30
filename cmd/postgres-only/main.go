package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"syncer-playground/pkg/chat"
	"syncer-playground/pkg/config"
	"syncer-playground/pkg/replication"
)

type server struct {
	chat.UnimplementedChatServiceServer
	db         *gorm.DB
	replicator *replication.PostgresReplicator
}

func (s *server) StreamDataChanges(req *chat.StreamDataChangesRequest, stream chat.ChatService_StreamDataChangesServer) error {
	// Create a channel for data change events
	eventChan := make(chan *chat.DataChangeEvent, 100)

	// Start replication
	if err := s.replicator.StartReplication(stream.Context(), eventChan); err != nil {
		return fmt.Errorf("failed to start replication: %w", err)
	}

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

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, &server{
		db:         db,
		replicator: replicator,
	})
	reflection.Register(s)

	log.Printf("Server listening on port %d", cfg.Server.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
