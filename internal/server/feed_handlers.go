package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/Takenobou/podconfig/web"
)

var tmpl *template.Template

func init() {
	fs, err := web.Templates()
	if err != nil {
		panic(err)
	}
	tmpl = template.Must(
		template.New("").Funcs(template.FuncMap{
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
		}).ParseFS(fs, "*.html"))
}

// Index handles the main page rendering.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	feedList, err := h.FeedService.GetFeedList(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error reading feed list: %v", err)
		feedList = []FeedListItem{}
	}
	data := map[string]interface{}{
		"Message":        r.URL.Query().Get("message"),
		"Feeds":          feedList,
		"PendingChanges": h.getChanges(),
	}
	if err := tmpl.ExecuteTemplate(w, "index", data); err != nil {
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
	options, err := parseFeedUpdate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	feed, err := h.YouTubeClient.FetchChannelInfo(r.Context(), youtubeUrl)
	if err != nil {
		log.Printf("Error fetching channel info: %v", err)
		http.Error(w, "Failed to fetch channel info", http.StatusInternalServerError)
		return
	}
	err = h.FeedService.AppendFeedToConfig(h.PodsyncConfigPath, feed, options)
	if errors.Is(err, ErrFeedExists) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, ErrInvalidFeedUpdate) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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
	tmpl.ExecuteTemplate(w, "index", data)
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
	err := h.FeedService.RemoveFeed(h.PodsyncConfigPath, feedKey)
	if errors.Is(err, ErrFeedNotFound) {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error removing feed: %v", err)
		http.Error(w, "Failed to remove feed", http.StatusInternalServerError)
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
	tmpl.ExecuteTemplate(w, "index", data)
}

// FeedListHandler returns the list of feeds in HTML (partial).
func (h *Handler) FeedListHandler(w http.ResponseWriter, r *http.Request) {
	feedList, err := h.FeedService.GetFeedList(h.PodsyncConfigPath)
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
	updates, err := parseFeedUpdate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.FeedService.ModifyFeed(h.PodsyncConfigPath, feedKey, updates)
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
	tmpl.ExecuteTemplate(w, "index", data)
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

// parseFeedUpdate converts HTTP form values; defaults and validation belong to the service.
func parseFeedUpdate(r *http.Request) (FeedUpdate, error) {
	var update FeedUpdate
	if value := r.FormValue("update_period"); value != "" {
		update.UpdatePeriod = &value
	}
	if value := r.FormValue("format"); value != "" {
		update.Format = &value
	}
	var err error
	update.CleanKeepLast, err = optionalFormInt(r, "clean_keep_last")
	if err != nil {
		return FeedUpdate{}, err
	}
	update.MaxAge, err = optionalFormInt(r, "max_age")
	if err != nil {
		return FeedUpdate{}, err
	}
	return update, update.validate()
}

func optionalFormInt(r *http.Request, name string) (*int, error) {
	value := r.FormValue(name)
	if value == "" {
		return nil, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s must be an integer", ErrInvalidFeedUpdate, name)
	}
	return &number, nil
}
