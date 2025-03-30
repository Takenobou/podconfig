package main

import (
	"log"
	"net/http"

	"github.com/Takenobou/podconfig/internal/config"
	"github.com/Takenobou/podconfig/internal/web"
)

func main() {
	cfg := config.LoadConfig()

	handler := &web.Handler{
		PodsyncConfigPath:   cfg.PodsyncConfigPath,
		DockerContainerName: cfg.DockerContainerName,
	}

	port := cfg.ServerPort

	http.HandleFunc("/", handler.Index)
	http.HandleFunc("/add", handler.AddFeedHandler)
	http.HandleFunc("/reload", handler.ReloadHandler)
	http.HandleFunc("/feeds", handler.FeedListHandler)

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
