package server

import (
	"context"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"time"
)

// HTMX stops polling when a response uses status 286.
const stopPolling = 286

// FeedXMLHandler enables copying once Podsync serves the configured RSS feed.
func (h *Handler) FeedXMLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	feeds, err := h.FeedService.GetFeedList(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error checking feed XML: %v", err)
		renderTemplate(w, http.StatusOK, "xmlCopy", FeedListItem{})
		return
	}
	for _, feed := range feeds {
		if feed.Key != r.URL.Query().Get("feedKey") {
			continue
		}
		feed.XMLAvailable = rssAvailable(r.Context(), feed.XMLURL)
		status := http.StatusOK
		if feed.XMLAvailable || feed.XMLURL == "" {
			status = stopPolling
		}
		renderTemplate(w, status, "xmlCopy", feed)
		return
	}
	// The feed may have been removed while an availability check was in flight.
	renderTemplate(w, stopPolling, "xmlCopy", FeedListItem{})
}

func rssAvailable(ctx context.Context, feedURL string) bool {
	if feedURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	// Inspect only the opening XML element, rather than downloading the whole feed.
	// A proxy's HTML error page must not enable the copy button, even with HTTP 200.
	decoder := xml.NewDecoder(io.LimitReader(resp.Body, 8<<10))
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if root, ok := token.(xml.StartElement); ok {
			return root.Name.Local == "rss"
		}
	}
}
