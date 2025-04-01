package web

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/pelletier/go-toml/v2"
)

// FeedService provides business logic for managing feeds.
type FeedService struct {
	mu sync.Mutex
}

// GetFeedList returns the list of feeds from the configuration file.
func (fs *FeedService) GetFeedList(configPath string) ([]FeedListItem, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		return []FeedListItem{}, nil
	}

	var hostname string
	if serverSection, ok := config["server"].(map[string]interface{}); ok {
		hostname, _ = serverSection["hostname"].(string)
	}

	var feedList []FeedListItem
	for key, v := range feeds {
		entry, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		// Determine feed name
		name := key
		if custom, ok := entry["custom"].(map[string]interface{}); ok {
			if t, ok := custom["title"].(string); ok && t != "" {
				name = t
			}
		}

		// Potential feed source URL
		urlVal, _ := entry["url"].(string)

		// Construct the feed’s XML URL, if hostname is configured
		xmlURL := ""
		if hostname != "" {
			xmlURL = strings.TrimRight(hostname, "/") + "/" + key + ".xml"
		}

		updatePeriod, _ := entry["update_period"].(string)
		format, _ := entry["format"].(string)

		// Retrieve max_age from filters
		var maxAge string
		if filters, ok := entry["filters"].(map[string]interface{}); ok {
			if ma, ok := filters["max_age"]; ok {
				maxAge = fmt.Sprintf("%v", ma)
			}
		}

		// Retrieve keep_last from clean
		var cleanKeepLast string
		if clean, ok := entry["clean"].(map[string]interface{}); ok {
			if ck, ok := clean["keep_last"]; ok {
				cleanKeepLast = fmt.Sprintf("%v", ck)
			}
		}

		feedList = append(feedList, FeedListItem{
			Key:           key,
			Name:          name,
			URL:           urlVal,
			XMLURL:        xmlURL,
			UpdatePeriod:  updatePeriod,
			Format:        format,
			MaxAge:        maxAge,
			CleanKeepLast: cleanKeepLast,
		})
	}

	sort.Slice(feedList, func(i, j int) bool {
		return feedList[i].Name < feedList[j].Name
	})

	return feedList, nil
}

// FetchChannelInfo retrieves channel info from the given YouTube URL.
func (fs *FeedService) FetchChannelInfo(youtubeUrl string) (*NewFeedInfo, error) {
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
	feedKey := Sanitise(channelName)

	return &NewFeedInfo{
		FeedKey:        feedKey,
		URL:            canonical,
		ChannelName:    channelName,
		ProfilePicture: profilePic,
	}, nil
}

// AppendFeedToConfig appends a new feed to the configuration.
func (fs *FeedService) AppendFeedToConfig(configPath string, feed *NewFeedInfo, updatePeriod string,
	feedFormat string, cleanKeepLast int, maxAge int) error {

	fs.mu.Lock()
	defer fs.mu.Unlock()

	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(content, &config); err != nil {
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
		"update_period":   updatePeriod,
		"quality":         "high",
		"format":          feedFormat,
		"opml":            true,
		"private_feed":    false,
		"youtube_dl_args": []string{"--add-metadata", "--embed-thumbnail", "--write-description"},
		"clean":           map[string]interface{}{"keep_last": cleanKeepLast},
		"filters":         map[string]interface{}{"max_age": maxAge},
		"custom": map[string]interface{}{
			"title":       feed.ChannelName,
			"description": "Episodes from the '" + feed.ChannelName + "' Youtube channel in a podcast format.",
			"author":      feed.ChannelName,
			"cover_art":   feed.ProfilePicture,
			"lang":        "en",
			"explicit":    false,
		},
	}

	feeds[feed.FeedKey] = newFeed
	newContent, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, newContent, 0644)
}

// ModifyFeed updates an existing feed's configuration with the provided updates.
func (fs *FeedService) ModifyFeed(configPath string, feedKey string, updates map[string]interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(content, &config); err != nil {
		return err
	}

	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no feeds found in config")
	}
	feed, exists := feeds[feedKey]
	if !exists {
		return fmt.Errorf("feed %s not found", feedKey)
	}
	feedMap, ok := feed.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid feed format")
	}

	// Merge each update into feedMap
	for key, value := range updates {
		feedMap[key] = value
	}

	newContent, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, newContent, 0644)
}

// Sanitise creates a feed key from the given channel name.
func Sanitise(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return strings.ToLower(sb.String())
}
