package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Takenobou/podconfig/internal/config"
	"github.com/Takenobou/podconfig/internal/server"
	"github.com/Takenobou/podconfig/web"
)

func main() {
	cfg := config.LoadConfig()

	feedService := &server.FeedService{}

	handler := &server.Handler{
		PodsyncConfigPath:   cfg.PodsyncConfigPath,
		DockerContainerName: cfg.DockerContainerName,
		FeedService:         feedService,
	}

	port := cfg.ServerPort

	staticFS, err := web.Static()
	if err != nil {
		log.Fatalf("Failed to load static assets: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	http.HandleFunc("/", handler.Index)
	http.HandleFunc("/add", handler.AddFeedHandler)
	http.HandleFunc("/reload", handler.ReloadHandler)
	http.HandleFunc("/feeds", handler.FeedListHandler)
	http.HandleFunc("/modify", handler.ModifyFeedHandler)
	http.HandleFunc("/remove", handler.RemoveFeedHandler)
	http.HandleFunc("/changelog", handler.ChangelogHandler)
	http.HandleFunc("/health", handler.HealthHandler)

	server := &http.Server{
		Addr: ":" + port,
	}

	go func() {
		log.Printf("Starting server on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
