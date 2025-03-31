package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func (h *Handler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	cmd := exec.Command("docker", "restart", h.DockerContainerName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		log.Printf("Error restarting container: %v, output: %s", err, out.String())
		http.Error(w, "Failed to reload docker container", http.StatusInternalServerError)
		return
	}
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
