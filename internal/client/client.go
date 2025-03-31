package client

import (
	"fmt"

	"github.com/jckhoe-sandbox/syncer-playground/internal/dep"
	"gorm.io/gorm"
)

type Client struct {
	db *gorm.DB
}

func NewClient(
	dbHost string,
	user string,
	password string,
	dbName string,
	port int,
) (*Client, error) {
	db, err := dep.NewPostgresDb(dbHost, user, password, dbName, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Client{
		db: db,
	}, nil
}

func (c *Client) Close() error {
	return nil
}

// func (c *Client) applyChange(event *chat.DataChangeEvent) error {
// 	switch event.Operation {
// 	case chat.Operation_OPERATION_INSERT:
// 		return c.db.Table(event.Table).Create(event.Data).Error
// 	case chat.Operation_OPERATION_UPDATE:
// 		return c.db.Table(event.Table).Where(event.OldData).Updates(event.Data).Error
// 	case chat.Operation_OPERATION_DELETE:
// 		return c.db.Table(event.Table).Delete(event.Data).Error
// 	default:
// 		return fmt.Errorf("unknown operation: %v", event.Operation)
// 	}
// }
