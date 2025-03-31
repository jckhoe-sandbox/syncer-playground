package main

import (
	"fmt"
	"log"
	"net"

	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	chat.UnimplementedChatServiceServer
	db *gorm.DB
}

func (s *server) StreamDataChanges(req *chat.StreamDataChangesRequest, stream chat.ChatService_StreamDataChangesServer) error {
	eventChan := make(chan *chat.DataChangeEvent, 100)

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
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.GetPostgresDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, &server{
		db: db,
	})
	reflection.Register(s)

	log.Printf("Server listening on port %d", cfg.Server.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
