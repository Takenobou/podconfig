package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ReloadHandler restarts the Docker container using the Docker API client.
func (h *Handler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Error creating Docker client: %v", err)
		http.Error(w, "Failed to create Docker client", http.StatusInternalServerError)
		return
	}

	timeout := 10 * time.Second
	ctx := context.Background()
	intTimeout := int(timeout.Seconds())
	err = cli.ContainerRestart(ctx, h.DockerContainerName, container.StopOptions{Timeout: &intTimeout})
	if err != nil {
		log.Printf("Error restarting container: %v", err)
		http.Error(w, "Failed to reload docker container", http.StatusInternalServerError)
		return
	}

	h.clearChanges()

	successMsg := fmt.Sprintf("Docker container '%s' reloaded successfully!", h.DockerContainerName)
	log.Printf("Reload successful: %s", successMsg)

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": successMsg})
		return
	}

	data := map[string]interface{}{
		"Message": successMsg,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
