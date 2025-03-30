package main

import (
	"context"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
)

func main() {
	serverAddr := flag.String("addr", "localhost:50051", "Server address")
	flag.Parse()

	// Connect to server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)
	ctx := context.Background()

	// Create a stream
	stream, err := client.ChatStream(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	// Start a goroutine to receive messages
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("Error receiving message: %v", err)
				return
			}
			log.Printf("Received message from %s: %s", msg.Sender, msg.Content)
		}
	}()

	// Send messages
	for i := 0; i < 5; i++ {
		msg := &pb.ChatMessage{
			Content:   "Hello from client!",
			Sender:    "Client",
			Timestamp: time.Now().Unix(),
		}

		if err := stream.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
			return
		}

		log.Printf("Sent message: %s", msg.Content)
		time.Sleep(time.Second)
	}

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Printf("Error closing stream: %v", err)
	}
}
