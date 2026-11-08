package server

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
)

// renderSuccess sends all updated regions in one response, or redirects a plain form.
func (h *Handler) renderSuccess(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/?"+url.Values{"message": {message}}.Encode(), http.StatusSeeOther)
		return
	}
	feeds, err := h.FeedService.GetFeedList(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error refreshing feed list: %v", err)
		h.renderError(w, r, "Saved. Refresh to update the feed list.", http.StatusInternalServerError)
		return
	}
	renderTemplate(w, http.StatusOK, "mutationResult", map[string]interface{}{
		"Message":        message,
		"Feeds":          feeds,
		"PendingChanges": h.getChanges(),
	})
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if r.Header.Get("HX-Request") != "true" {
		http.Error(w, message, status)
		return
	}
	// Errors update only the message: entered values and open edit forms stay intact.
	renderTemplate(w, status, "message", map[string]interface{}{"Message": message})
}

func renderTemplate(w http.ResponseWriter, status int, name string, data interface{}) {
	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, name, data); err != nil {
		log.Printf("Error rendering %s: %v", name, err)
		http.Error(w, "Could not load the page.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = body.WriteTo(w)
}
