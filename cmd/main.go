package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Takenobou/podconfig/internal/config"
	"github.com/Takenobou/podconfig/internal/web"
)

func main() {
	appConfigPath := os.Getenv("APPCONFIG_PATH")
	if appConfigPath == "" {
		appConfigPath = "config/appconfig.toml"
	}
	cfg, err := config.LoadConfig(appConfigPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	handler := &web.Handler{
		PodsyncConfigPath:   cfg.PodsyncConfigPath,
		DockerContainerName: cfg.DockerContainerName,
	}

	port := cfg.ServerPort
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handler.Index)
	http.HandleFunc("/add", handler.AddFeedHandler)
	http.HandleFunc("/reload", handler.ReloadHandler)

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
