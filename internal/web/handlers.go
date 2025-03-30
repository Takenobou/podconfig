package web

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pelletier/go-toml/v2"
)

//go:embed templates/index.html
var indexHTML string

var tmpl = template.Must(template.New("index").Parse(indexHTML))

type FeedInfo struct {
	FeedKey        string
	URL            string
	ChannelName    string
	ProfilePicture string
}

type Handler struct {
	PodsyncConfigPath   string
	DockerContainerName string
}

// Index renders the main page with no special data.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

// AddFeedHandler handles form submission: reads the YouTube URL, fetches channel info,
// and appends the new feed entry to the Podsync config.
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
	feed, err := fetchChannelInfo(youtubeUrl)
	if err != nil {
		log.Printf("Error fetching channel info: %v", err)
		http.Error(w, "Failed to fetch channel info", http.StatusInternalServerError)
		return
	}
	err = appendFeedToConfig(h.PodsyncConfigPath, feed)
	if err != nil {
		log.Printf("Error updating config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Message": fmt.Sprintf("Feed for channel '%s' added successfully!", feed.ChannelName),
	}
	tmpl.Execute(w, data)
}

// ReloadHandler executes "docker restart" for the container specified in the config.
func (h *Handler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	data := map[string]interface{}{
		"Message": fmt.Sprintf("Docker container '%s' reloaded successfully!", h.DockerContainerName),
	}
	tmpl.Execute(w, data)
}

// fetchChannelInfo fetches the provided YouTube URL and extracts channel details.
func fetchChannelInfo(youtubeUrl string) (*FeedInfo, error) {
	resp, err := http.Get(youtubeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract the canonical link
	canonical, exists := doc.Find("link[rel='canonical']").Attr("href")
	if !exists || canonical == "" {
		return nil, fmt.Errorf("canonical link not found")
	}

	// If the canonical link doesn't contain /channel/, look for meta itemprop=channelId
	if !strings.Contains(canonical, "/channel/") {
		channelId, exists := doc.Find("meta[itemprop='channelId']").Attr("content")
		if !exists || channelId == "" {
			return nil, fmt.Errorf("channel id not found")
		}
		canonical = "https://www.youtube.com/channel/" + channelId
	}

	channelName, exists := doc.Find("meta[property='og:title']").Attr("content")
	if !exists || channelName == "" {
		channelName = "Unknown Channel"
	}

	profilePic, _ := doc.Find("meta[property='og:image']").Attr("content")
	feedKey := sanitise(channelName)

	return &FeedInfo{
		FeedKey:        feedKey,
		URL:            canonical,
		ChannelName:    channelName,
		ProfilePicture: profilePic,
	}, nil
}

// appendFeedToConfig loads the Podsync config from configPath, adds (or updates) the feed,
// and writes the file back.
func appendFeedToConfig(configPath string, feed *FeedInfo) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	err = toml.Unmarshal(content, &config)
	if err != nil {
		return err
	}

	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		feeds = make(map[string]interface{})
		config["feeds"] = feeds
	}

	// Build the new feed entry
	newFeed := map[string]interface{}{
		"url":             feed.URL,
		"page_size":       50,
		"update_period":   "6h",
		"quality":         "high",
		"format":          "video",
		"opml":            true,
		"cron_schedule":   "@every 6h",
		"private_feed":    false,
		"youtube_dl_args": []string{"--add-metadata", "--embed-thumbnail", "--write-description"},
		"clean":           map[string]interface{}{"keep_last": 20},
		"filters":         map[string]interface{}{"max_age": 90},
		"custom": map[string]interface{}{
			"title":         feed.ChannelName,
			"description":   "Episodes from '" + feed.ChannelName + "' channel in podcast format.",
			"author":        feed.ChannelName,
			"cover_art":     feed.ProfilePicture,
			"lang":          "en",
			"category":      "News",
			"subcategories": []string{"Politics", "Commentary"},
			"explicit":      false,
		},
	}
	feeds[feed.FeedKey] = newFeed

	newContent, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, newContent, 0644)
}

// sanitise creates a feed key from the channel name by removing non-alphanumerics
// and converting to lower-case.
func sanitise(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return strings.ToLower(sb.String())
}
