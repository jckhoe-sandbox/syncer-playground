package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/jckhoe-sandbox/syncer-playground/internal/dep"
	"github.com/jckhoe-sandbox/syncer-playground/internal/server"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	InitConfig()

	db, err := dep.NewPostgresDb(config.Db.Hostname, config.Db.Username, config.Db.Password, config.Db.DbName, config.Db.Port)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	eventManager, err := events.NewRedisEventManager(1, config.Redis.Port, config.Redis.Hostname, config.Redis.Password)
	if err != nil {
		log.Fatalf("Failed to create event manager: %v", err)
	}
	defer eventManager.Close()

	srv := server.NewServer(db, eventManager)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.StartRedisSubscriber(ctx); err != nil {
		log.Fatalf("Failed to start Redis subscriber: %v", err)
	}

	if err := srv.StartPostgresReplicator(ctx); err != nil {
		log.Fatalf("Failed to start PostgreSQL replicator: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Http.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, srv)
	reflection.Register(s)

	log.Printf("Server listening on port %d", config.Http.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
