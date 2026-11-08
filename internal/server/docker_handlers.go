package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/moby/moby/client"
)

// ReloadHandler restarts the Docker container using the Docker API client.
func (h *Handler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, r, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	cli, err := client.New(client.FromEnv)
	if err != nil {
		log.Printf("Error creating Docker client: %v", err)
		h.renderError(w, r, "Could not reload Podsync.", http.StatusInternalServerError)
		return
	}

	defer cli.Close()

	timeout := 10 * time.Second
	ctx := context.Background()
	intTimeout := int(timeout.Seconds())
	_, err = cli.ContainerRestart(ctx, h.DockerContainerName, client.ContainerRestartOptions{Timeout: &intTimeout})
	if err != nil {
		log.Printf("Error restarting container: %v", err)
		h.renderError(w, r, "Could not reload Podsync.", http.StatusInternalServerError)
		return
	}

	h.clearChanges()

	successMsg := "Podsync reloaded."
	log.Printf("Reloaded container %s", h.DockerContainerName)

	if r.Header.Get("HX-Request") == "true" {
		// Reloading does not change feeds. Preserve their current copy and edit state.
		renderTemplate(w, http.StatusOK, "reloadResult", map[string]interface{}{
			"Message":        successMsg,
			"PendingChanges": h.getChanges(),
		})
		return
	}
	h.renderSuccess(w, r, successMsg)
}
