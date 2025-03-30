package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jckhoe-sandbox/syncer-playground/internal/client"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: client <username>")
	}
	username := os.Args[1]

	chatClient, err := client.NewChatClient("localhost:50051", username)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer chatClient.Close()

	if err := chatClient.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	go chatClient.ReceiveMessages()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Connected as %s. Type messages (press Ctrl+C to exit):\n", username)

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down...")
			return
		default:
			if scanner.Scan() {
				message := scanner.Text()
				if err := chatClient.SendMessage(message); err != nil {
					log.Printf("Failed to send message: %v", err)
				}
			}
		}
	}
}
