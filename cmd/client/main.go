package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jckhoe-sandbox/syncer-playground/pkg/chat"
	"github.com/jckhoe-sandbox/syncer-playground/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	db          *gorm.DB
	pgOnlyConn  *grpc.ClientConn
	pgRedisConn *grpc.ClientConn
	pgOnlyCli   chat.ChatServiceClient
	pgRedisCli  chat.ChatServiceClient
}

func NewClient(cfg *config.Config) (*Client, error) {
	db, err := gorm.Open(postgres.Open(cfg.GetPostgresDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	pgOnlyConn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL-only server: %w", err)
	}

	pgRedisConn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		pgOnlyConn.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL + Redis server: %w", err)
	}

	return &Client{
		db:          db,
		pgOnlyConn:  pgOnlyConn,
		pgRedisConn: pgRedisConn,
		pgOnlyCli:   chat.NewChatServiceClient(pgOnlyConn),
		pgRedisCli:  chat.NewChatServiceClient(pgRedisConn),
	}, nil
}

func (c *Client) Close() error {
	var errs []error
	if err := c.pgOnlyConn.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close PostgreSQL-only connection: %w", err))
	}
	if err := c.pgRedisConn.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close PostgreSQL + Redis connection: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

func (c *Client) StreamChanges(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		stream, err := c.pgOnlyCli.StreamDataChanges(ctx, &chat.StreamDataChangesRequest{})
		if err != nil {
			errChan <- fmt.Errorf("failed to start streaming from PostgreSQL-only server: %w", err)
			return
		}

		for {
			event, err := stream.Recv()
			if err != nil {
				errChan <- fmt.Errorf("error receiving from PostgreSQL-only server: %w", err)
				return
			}

			log.Printf("Received event from PostgreSQL-only server: %v", event)
			if err := c.applyChange(event); err != nil {
				log.Printf("Error applying change from PostgreSQL-only server: %v", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		stream, err := c.pgRedisCli.StreamDataChanges(ctx, &chat.StreamDataChangesRequest{})
		if err != nil {
			errChan <- fmt.Errorf("failed to start streaming from PostgreSQL + Redis server: %w", err)
			return
		}

		for {
			event, err := stream.Recv()
			if err != nil {
				errChan <- fmt.Errorf("error receiving from PostgreSQL + Redis server: %w", err)
				return
			}

			log.Printf("Received event from PostgreSQL + Redis server: %v", event)
			if err := c.applyChange(event); err != nil {
				log.Printf("Error applying change from PostgreSQL + Redis server: %v", err)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) applyChange(event *chat.DataChangeEvent) error {
	switch event.Operation {
	case chat.Operation_OPERATION_INSERT:
		return c.db.Table(event.Table).Create(event.Data).Error
	case chat.Operation_OPERATION_UPDATE:
		return c.db.Table(event.Table).Where(event.OldData).Updates(event.Data).Error
	case chat.Operation_OPERATION_DELETE:
		return c.db.Table(event.Table).Delete(event.Data).Error
	default:
		return fmt.Errorf("unknown operation: %v", event.Operation)
	}
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.StreamChanges(context.Background()); err != nil {
		log.Fatalf("Error streaming changes: %v", err)
	}
}
