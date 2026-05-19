package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mini-redis/internal/server"
)

func main() {
	fmt.Println("[Mini-Redis] Starting on port 6379...")

	srv := server.New(":6379")

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Println("Server error:", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[Mini-Redis] Shutting down...")
	srv.Stop()
}
