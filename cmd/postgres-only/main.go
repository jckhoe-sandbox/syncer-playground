package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/config"
)

type server struct {
	pb.UnimplementedChatServiceServer
	db *gorm.DB
}

func (s *server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		// Save message to database
		if err := s.db.Create(msg).Error; err != nil {
			log.Printf("Error saving message: %v", err)
			continue
		}

		// Send acknowledgment back to client
		if err := stream.Send(msg); err != nil {
			log.Printf("Error sending response: %v", err)
			return err
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
	db, err := gorm.Open(postgres.Open(cfg.Postgres.GetDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&pb.ChatMessage{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server{db: db})

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
