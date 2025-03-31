package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jckhoe-sandbox/syncer-playground/internal/client"
)

func main() {
	InitConfig()

	client, err := client.NewClient(config.Db.Hostname, config.Db.Username, config.Db.Password, config.Db.DbName, config.Db.Port)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))
	router.Use(gin.Recovery())

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
}
