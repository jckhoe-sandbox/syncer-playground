package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/config"
)

type server struct {
	pb.UnimplementedChatServiceServer
	db    *gorm.DB
	redis *redis.Client
}

func (s *server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	ctx := context.Background()

	// Subscribe to Redis channel
	pubsub := s.redis.Subscribe(ctx, "chat_messages")
	defer pubsub.Close()

	// Start a goroutine to handle Redis messages
	go func() {
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Printf("Error receiving Redis message: %v", err)
				continue
			}

			var chatMsg pb.ChatMessage
			if err := json.Unmarshal([]byte(msg.Payload), &chatMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			// Save message to database
			if err := s.db.Create(&chatMsg).Error; err != nil {
				log.Printf("Error saving message from Redis: %v", err)
				continue
			}

			// Send message to client
			if err := stream.Send(&chatMsg); err != nil {
				log.Printf("Error sending message to client: %v", err)
			}
		}
	}()

	// Handle incoming gRPC messages
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

		// Publish message to Redis
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}

		if err := s.redis.Publish(ctx, "chat_messages", msgJSON).Err(); err != nil {
			log.Printf("Error publishing to Redis: %v", err)
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

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server{
		db:    db,
		redis: rdb,
	})

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
} 