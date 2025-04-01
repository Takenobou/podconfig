package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

var fs = &FeedService{}
var tmpl = template.Must(template.New("index").Funcs(template.FuncMap{
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict expects an even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	},
}).Parse(indexHTML))

// NewFeedInfo holds information about a feed.
type NewFeedInfo struct {
	FeedKey        string
	URL            string
	ChannelName    string
	ProfilePicture string
}

// FeedListItem represents an entry in the feed list.
type FeedListItem struct {
	Key           string
	Name          string
	URL           string
	XMLURL        string
	UpdatePeriod  string
	Format        string
	MaxAge        string
	CleanKeepLast string
}

// Index handles the main page rendering.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	feedList, err := fs.GetFeedList(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error reading feed list: %v", err)
		feedList = []FeedListItem{}
	}
	data := map[string]interface{}{
		"Message":        r.URL.Query().Get("message"),
		"Feeds":          feedList,
		"PendingChanges": h.getChanges(), // pass the pending changes to the template
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// AddFeedHandler handles adding a new feed.
func (h *Handler) AddFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	youtubeUrl := r.FormValue("youtubeUrl")
	if youtubeUrl == "" {
		http.Error(w, "YouTube URL is required", http.StatusBadRequest)
		return
	}
	updatePeriod := r.FormValue("update_period")
	if updatePeriod == "" {
		updatePeriod = "1h"
	}
	feedFormat := r.FormValue("format")
	if feedFormat == "" {
		feedFormat = "video"
	}
	cleanKeepLast := 20
	if val := r.FormValue("clean_keep_last"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cleanKeepLast = v
		}
	}
	maxAge := 90
	if val := r.FormValue("max_age"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			maxAge = v
		}
	}
	feed, err := fs.FetchChannelInfo(youtubeUrl)
	if err != nil {
		log.Printf("Error fetching channel info: %v", err)
		http.Error(w, "Failed to fetch channel info", http.StatusInternalServerError)
		return
	}
	err = fs.AppendFeedToConfig(h.PodsyncConfigPath, feed, updatePeriod, feedFormat, cleanKeepLast, maxAge)
	if err != nil {
		log.Printf("Error updating config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Added feed '%s'", feed.FeedKey))

	successMsg := fmt.Sprintf("Feed for channel '%s' added successfully!", feed.ChannelName)
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": successMsg})
		return
	}
	data := map[string]interface{}{
		"Message": successMsg,
	}
	tmpl.Execute(w, data)
}

// RemoveFeedHandler handles removing a feed.
func (h *Handler) RemoveFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	feedKey := r.FormValue("feedKey")
	if feedKey == "" {
		http.Error(w, "feedKey is required", http.StatusBadRequest)
		return
	}
	content, err := os.ReadFile(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error reading config: %v", err)
		http.Error(w, "Failed to read config", http.StatusInternalServerError)
		return
	}
	var config map[string]interface{}
	err = toml.Unmarshal(content, &config)
	if err != nil {
		log.Printf("Error unmarshalling config: %v", err)
		http.Error(w, "Failed to parse config", http.StatusInternalServerError)
		return
	}
	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		http.Error(w, "No feeds found", http.StatusInternalServerError)
		return
	}
	if _, exists := feeds[feedKey]; !exists {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}
	delete(feeds, feedKey)
	config["feeds"] = feeds
	newContent, err := toml.Marshal(config)
	if err != nil {
		log.Printf("Error marshalling config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(h.PodsyncConfigPath, newContent, 0644)
	if err != nil {
		log.Printf("Error writing config: %v", err)
		http.Error(w, "Failed to write config", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Removed feed '%s'", feedKey))

	successMsg := fmt.Sprintf("Feed for channel '%s' removed successfully!", feedKey)
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": successMsg})
		return
	}
	data := map[string]interface{}{
		"Message": successMsg,
	}
	tmpl.Execute(w, data)
}

// FeedListHandler returns the list of feeds in HTML (partial).
func (h *Handler) FeedListHandler(w http.ResponseWriter, r *http.Request) {
	feedList, err := fs.GetFeedList(h.PodsyncConfigPath)
	if err != nil {
		http.Error(w, "Failed to load feed list", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Feeds": feedList,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "feedList", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// ModifyFeedHandler handles updating an existing feed.
func (h *Handler) ModifyFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	feedKey := r.FormValue("feedKey")
	if feedKey == "" {
		http.Error(w, "feedKey is required", http.StatusBadRequest)
		return
	}
	updates := make(map[string]interface{})
	if updatePeriod := r.FormValue("update_period"); updatePeriod != "" {
		updates["update_period"] = updatePeriod
	}
	if feedFormat := r.FormValue("format"); feedFormat != "" {
		updates["format"] = feedFormat
	}
	if cleanKeepLastStr := r.FormValue("clean_keep_last"); cleanKeepLastStr != "" {
		if v, err := strconv.Atoi(cleanKeepLastStr); err == nil {
			updates["clean"] = map[string]interface{}{"keep_last": v}
		}
	}
	if maxAgeStr := r.FormValue("max_age"); maxAgeStr != "" {
		if v, err := strconv.Atoi(maxAgeStr); err == nil {
			updates["filters"] = map[string]interface{}{"max_age": v}
		}
	}
	err := fs.ModifyFeed(h.PodsyncConfigPath, feedKey, updates)
	if err != nil {
		log.Printf("Error modifying feed: %v", err)
		http.Error(w, "Failed to modify feed", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Modified feed '%s'", feedKey))

	successMsg := fmt.Sprintf("Feed '%s' modified successfully!.", feedKey)
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": successMsg})
		return
	}
	data := map[string]interface{}{
		"Message": successMsg,
	}
	tmpl.Execute(w, data)
}

// ChangelogHandler returns the minimal changelog partial
func (h *Handler) ChangelogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	data := map[string]interface{}{
		"PendingChanges": h.getChanges(),
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "changelogOnly", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
