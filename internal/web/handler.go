package web

import "sync"

// Handler is the HTTP handler for podconfig.
type Handler struct {
	PodsyncConfigPath   string
	DockerContainerName string

	mu      sync.Mutex
	pending []string //
}

// addChange records a message in the pending changelog.
func (h *Handler) addChange(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = append(h.pending, msg)
}

// clearChanges removes all pending changes (after a successful container reload).
func (h *Handler) clearChanges() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = nil
}

// getChanges returns a snapshot of the current pending changes.
func (h *Handler) getChanges() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	cpy := make([]string, len(h.pending))
	copy(cpy, h.pending)
	return cpy
}
