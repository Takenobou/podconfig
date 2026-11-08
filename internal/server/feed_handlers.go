package server

import (
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
		h.renderError(w, r, "Enter a YouTube URL.", http.StatusBadRequest)
		return
	}
	options, err := parseFeedUpdate(r)
	if err != nil {
		h.renderError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	feed, err := h.YouTubeClient.FetchChannelInfo(r.Context(), youtubeUrl)
	if err != nil {
		log.Printf("Error fetching channel info: %v", err)
		h.renderError(w, r, "Could not load the channel.", http.StatusInternalServerError)
		return
	}
	err = h.FeedService.AppendFeedToConfig(h.PodsyncConfigPath, feed, options)
	if errors.Is(err, ErrFeedExists) {
		h.renderError(w, r, "Feed already exists.", http.StatusConflict)
		return
	}
	if errors.Is(err, ErrInvalidFeedUpdate) {
		h.renderError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Error updating config: %v", err)
		h.renderError(w, r, "Could not save the feed.", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Added %s", feed.FeedKey))

	successMsg := "Feed added."
	h.renderSuccess(w, r, successMsg)
}

// RemoveFeedHandler handles removing a feed.
func (h *Handler) RemoveFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, r, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	feedKey := r.FormValue("feedKey")
	if feedKey == "" {
		h.renderError(w, r, "Feed not found.", http.StatusBadRequest)
		return
	}
	err := h.FeedService.RemoveFeed(h.PodsyncConfigPath, feedKey)
	if errors.Is(err, ErrFeedNotFound) {
		h.renderError(w, r, "Feed not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error removing feed: %v", err)
		h.renderError(w, r, "Could not remove the feed.", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Removed %s", feedKey))

	successMsg := "Feed removed."
	h.renderSuccess(w, r, successMsg)
}

// ModifyFeedHandler handles updating an existing feed.
func (h *Handler) ModifyFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, r, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	feedKey := r.FormValue("feedKey")
	if feedKey == "" {
		h.renderError(w, r, "Feed not found.", http.StatusBadRequest)
		return
	}
	updates, err := parseFeedUpdate(r)
	if err != nil {
		h.renderError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.FeedService.ModifyFeed(h.PodsyncConfigPath, feedKey, updates)
	if err != nil {
		log.Printf("Error modifying feed: %v", err)
		h.renderError(w, r, "Could not save the feed.", http.StatusInternalServerError)
		return
	}

	h.addChange(fmt.Sprintf("Updated %s", feedKey))

	successMsg := "Feed saved."
	h.renderSuccess(w, r, successMsg)
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
	update.CleanKeepLast, err = optionalFormInt(r, "clean_keep_last", "episode limit")
	if err != nil {
		return FeedUpdate{}, err
	}
	update.MaxAge, err = optionalFormInt(r, "max_age", "max age")
	if err != nil {
		return FeedUpdate{}, err
	}
	return update, update.validate()
}

func optionalFormInt(r *http.Request, name, label string) (*int, error) {
	value := r.FormValue(name)
	if value == "" {
		return nil, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s must be a whole number.", ErrInvalidFeedUpdate, label)
	}
	return &number, nil
}
