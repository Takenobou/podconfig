package web

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
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

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	feedList, err := getFeedList(h.PodsyncConfigPath)
	if err != nil {
		log.Printf("Error reading feed list: %v", err)
		feedList = []FeedListItem{}
	}
	data := map[string]interface{}{
		"Message": r.URL.Query().Get("message"),
		"Feeds":   feedList,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// AddFeedHandler handles the form submission to add a new feed.
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

// ReloadHandler executes "docker restart" for the configured container.
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
	tmpl.Execute(w, data)
}

// FeedListHandler returns a JSON array of current feed names.
func (h *Handler) FeedListHandler(w http.ResponseWriter, r *http.Request) {
	feedList, err := getFeedList(h.PodsyncConfigPath)
	if err != nil {
		http.Error(w, "Failed to load feed list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feedList)
}

// FeedListItem represents an entry in the feed list.
type FeedListItem struct {
	Name   string
	URL    string
	XMLURL string
}

func getFeedList(configPath string) ([]FeedListItem, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		return []FeedListItem{}, nil
	}

	// Get the server hostname from the [server] section.
	var hostname string
	if serverSection, ok := config["server"].(map[string]interface{}); ok {
		hostname, _ = serverSection["hostname"].(string)
	}

	feedList := make([]FeedListItem, 0, len(feeds))
	for key, v := range feeds {
		entry, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		// Default title is the feed key.
		name := key
		// If a custom title is set, use it.
		if custom, ok := entry["custom"].(map[string]interface{}); ok {
			if t, ok := custom["title"].(string); ok && t != "" {
				name = t
			}
		}
		urlVal, _ := entry["url"].(string)
		xmlURL := ""
		if hostname != "" {
			xmlURL = strings.TrimRight(hostname, "/") + "/" + key + ".xml"
		}
		feedList = append(feedList, FeedListItem{
			Name:   name,
			URL:    urlVal,
			XMLURL: xmlURL,
		})
	}

	// Sort the feed list alphabetically by feed Name.
	sort.Slice(feedList, func(i, j int) bool {
		return feedList[i].Name < feedList[j].Name
	})

	return feedList, nil
}

// fetchChannelInfo retrieves the channel info from the YouTube URL.
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
	canonical, exists := doc.Find("link[rel='canonical']").Attr("href")
	if !exists || canonical == "" {
		return nil, fmt.Errorf("canonical link not found")
	}
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

// appendFeedToConfig updates the config file with the new feed.
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
	newFeed := map[string]interface{}{
		"url":             feed.URL,
		"page_size":       50,
		"update_period":   "1h",
		"quality":         "high",
		"format":          "video",
		"opml":            true,
		"cron_schedule":   "@every 1h",
		"private_feed":    false,
		"youtube_dl_args": []string{"--add-metadata", "--embed-thumbnail", "--write-description"},
		"clean":           map[string]interface{}{"keep_last": 20},
		"filters":         map[string]interface{}{"max_age": 90},
		"custom": map[string]interface{}{
			"title":         feed.ChannelName,
			"description":   "Episodes from the '" + feed.ChannelName + "' Youtube channel in a podcast format.",
			"author":        feed.ChannelName,
			"cover_art":     feed.ProfilePicture,
			"lang":          "en",
			"category":      "Entertainment",
			"subcategories": []string{"Commentary"},
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

// sanitise creates a feed key from the channel name.
func sanitise(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return strings.ToLower(sb.String())
}
