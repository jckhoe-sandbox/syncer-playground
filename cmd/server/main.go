package main

import (
	"log"
	"net"

	"github.com/jckhoe-sandbox/syncer-playground/internal/server"
	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, server.NewChatServer())
	log.Printf("Server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
