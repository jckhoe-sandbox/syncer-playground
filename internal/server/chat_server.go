package server

import (
	"context"
	"io"
	"log"
	"sync"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
)

type ChatServer struct {
	pb.UnimplementedChatServiceServer
	mu       sync.RWMutex
	clients  map[string]pb.ChatService_ChatStreamServer
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[string]pb.ChatService_ChatStreamServer),
	}
}

func (s *ChatServer) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	// Handle incoming messages
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			// Client disconnected
			s.mu.Lock()
			delete(s.clients, msg.Sender)
			s.mu.Unlock()
			return nil
		}
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			return err
		}

		// Store client stream if not already stored
		s.mu.Lock()
		if _, exists := s.clients[msg.Sender]; !exists {
			s.clients[msg.Sender] = stream
		}
		s.mu.Unlock()

		// Broadcast message to all other clients
		s.mu.RLock()
		for clientID, clientStream := range s.clients {
			if clientID != msg.Sender {
				if err := clientStream.Send(msg); err != nil {
					log.Printf("Error sending message to client %s: %v", clientID, err)
				}
			}
		}
		s.mu.RUnlock()
	}
} 