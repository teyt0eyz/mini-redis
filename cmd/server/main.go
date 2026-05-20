package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mini-redis/internal/metrics"
	"mini-redis/internal/server"
)

func main() {
	port := flag.String("port", "6379", "server port")
	metricsPort := flag.String("metrics", "2112", "metrics port")
	flag.Parse()

	metrics.Start(":" + *metricsPort)
	fmt.Printf("[Mini-Redis] Metrics at http://localhost:%s/metrics\n", *metricsPort)

	srv := server.New(":" + *port)

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
