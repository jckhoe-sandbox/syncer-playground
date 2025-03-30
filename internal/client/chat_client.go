package client

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"google.golang.org/grpc"
)

type ChatClient struct {
	conn     *grpc.ClientConn
	client   pb.ChatServiceClient
	stream   pb.ChatService_ChatStreamClient
	username string
	mu       sync.Mutex
}

func NewChatClient(serverAddr, username string) (*ChatClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewChatServiceClient(conn)
	return &ChatClient{
		conn:     conn,
		client:   client,
		username: username,
	}, nil
}

func (c *ChatClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	stream, err := c.client.ChatStream(context.Background())
	if err != nil {
		return err
	}

	c.stream = stream
	return nil
}

func (c *ChatClient) SendMessage(content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stream == nil {
		return io.EOF
	}

	msg := &pb.ChatMessage{
		Content:   content,
		Sender:    c.username,
		Timestamp: time.Now().Unix(),
	}

	return c.stream.Send(msg)
}

func (c *ChatClient) ReceiveMessages() {
	for {
		msg, err := c.stream.Recv()
		if err == io.EOF {
			log.Printf("Stream closed")
			return
		}
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			return
		}

		log.Printf("Received message from %s: %s", msg.Sender, msg.Content)
	}
}

func (c *ChatClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stream != nil {
		if err := c.stream.CloseSend(); err != nil {
			return err
		}
	}
	return c.conn.Close()
} 