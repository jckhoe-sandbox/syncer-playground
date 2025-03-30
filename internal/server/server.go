package server

import (
	"context"
	"log"
	"sync"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jackc/pgx/v5"
)

type ChatServer struct {
	pb.UnimplementedChatServiceServer
	mu       sync.RWMutex
	clients  map[string]pb.ChatService_StreamChangesServer
	pgConn   *pgx.Conn
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[string]pb.ChatService_StreamChangesServer),
	}
}

func (s *ChatServer) StreamChanges(req *pb.StreamRequest, stream pb.ChatService_StreamChangesServer) error {
	clientID := req.GetClientId()
	log.Printf("New client connected: %s", clientID)

	s.mu.Lock()
	s.clients[clientID] = stream
	s.mu.Unlock()

	// Keep the stream open
	<-stream.Context().Done()

	s.mu.Lock()
	delete(s.clients, clientID)
	s.mu.Unlock()

	log.Printf("Client disconnected: %s", clientID)
	return nil
}

func (s *ChatServer) broadcastChange(change *pb.Change) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for clientID, stream := range s.clients {
		if err := stream.Send(change); err != nil {
			log.Printf("Failed to send change to client %s: %v", clientID, err)
		}
	}
}

func (s *ChatServer) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	return &pb.ConnectResponse{
		Success: true,
		Message: "Successfully connected to the server",
	}, nil
} 