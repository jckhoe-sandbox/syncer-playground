package main

import (
	"fmt"
	"log"
	"net"

	"github.com/jckhoe-sandbox/syncer-playground/internal/dep"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	InitConfig()

	db, err := dep.NewPostgresDb(config.Db.Hostname, config.Db.Username, config.Db.Password, config.Db.DbName, config.Db.Port)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Http.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, &server{
		db: db,
	})
	reflection.Register(s)

	log.Printf("Server listening on port %d", config.Http.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
