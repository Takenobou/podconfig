package server

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
)

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
	XMLAvailable  bool
	UpdatePeriod  string
	Format        string
	MaxAge        string
	CleanKeepLast string
}

// FeedUpdate contains only the settings managed by Podconfig. Nil fields are unchanged.
type FeedUpdate struct {
	UpdatePeriod  *string
	Format        *string
	CleanKeepLast *int
	MaxAge        *int
}

var ErrInvalidFeedUpdate = errors.New("Invalid settings")

func (u FeedUpdate) validate() error {
	if u.UpdatePeriod != nil {
		duration, err := time.ParseDuration(*u.UpdatePeriod)
		if err != nil || duration <= 0 {
			return fmt.Errorf("%w: update interval must be positive (e.g. 1h).", ErrInvalidFeedUpdate)
		}
	}
	if u.Format != nil && *u.Format != "audio" && *u.Format != "video" {
		return fmt.Errorf("%w: format must be audio or video.", ErrInvalidFeedUpdate)
	}
	if u.CleanKeepLast != nil && *u.CleanKeepLast < 0 {
		return fmt.Errorf("%w: episode limit cannot be negative.", ErrInvalidFeedUpdate)
	}
	if u.MaxAge != nil && *u.MaxAge < 0 {
		return fmt.Errorf("%w: max age cannot be negative.", ErrInvalidFeedUpdate)
	}
	return nil
}

// FeedService provides business logic for managing feeds.
type FeedService struct {
	mu sync.Mutex
}

// GetFeedList returns the list of feeds from the configuration file.
func (fs *FeedService) GetFeedList(configPath string) ([]FeedListItem, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	config, err := readConfig(configPath)
	if err != nil {
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

// AppendFeedToConfig appends a new feed to the configuration.
func (fs *FeedService) AppendFeedToConfig(configPath string, feed *NewFeedInfo, options FeedUpdate) error {
	if err := options.validate(); err != nil {
		return err
	}
	updatePeriod, feedFormat, cleanKeepLast, maxAge := "1h", "video", 20, 90
	if options.UpdatePeriod != nil {
		updatePeriod = *options.UpdatePeriod
	}
	if options.Format != nil {
		feedFormat = *options.Format
	}
	if options.CleanKeepLast != nil {
		cleanKeepLast = *options.CleanKeepLast
	}
	if options.MaxAge != nil {
		maxAge = *options.MaxAge
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	config, err := readConfig(configPath)
	if err != nil {
		return err
	}

	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		feeds = make(map[string]interface{})
		config["feeds"] = feeds
	}

	if feed.FeedKey == "" {
		return fmt.Errorf("%w: channel name needs letters or numbers.", ErrInvalidFeedUpdate)
	}
	if _, exists := feeds[feed.FeedKey]; exists {
		return fmt.Errorf("%w: key %s", ErrFeedExists, feed.FeedKey)
	}
	for _, value := range feeds {
		existing, ok := value.(map[string]interface{})
		if ok && feed.URL != "" && existing["url"] == feed.URL {
			return fmt.Errorf("%w: channel %s", ErrFeedExists, feed.ChannelName)
		}
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
	return writeConfig(configPath, config)
}

// ModifyFeed updates an existing feed's configuration with the provided updates.
func (fs *FeedService) ModifyFeed(configPath string, feedKey string, updates FeedUpdate) error {
	if err := updates.validate(); err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()

	config, err := readConfig(configPath)
	if err != nil {
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

	if updates.UpdatePeriod != nil {
		feedMap["update_period"] = *updates.UpdatePeriod
	}
	if updates.Format != nil {
		feedMap["format"] = *updates.Format
	}
	if updates.CleanKeepLast != nil {
		clean, ok := feedMap["clean"].(map[string]interface{})
		if !ok {
			clean = make(map[string]interface{})
			feedMap["clean"] = clean
		}
		clean["keep_last"] = *updates.CleanKeepLast
	}
	if updates.MaxAge != nil {
		filters, ok := feedMap["filters"].(map[string]interface{})
		if !ok {
			filters = make(map[string]interface{})
			feedMap["filters"] = filters
		}
		filters["max_age"] = *updates.MaxAge
	}

	return writeConfig(configPath, config)
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

var ErrFeedNotFound = errors.New("feed not found")
var ErrFeedExists = errors.New("feed already exists")

// RemoveFeed shares the same lock as all other configuration operations.
func (fs *FeedService) RemoveFeed(configPath, feedKey string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}
	feeds, ok := config["feeds"].(map[string]interface{})
	if !ok {
		return ErrFeedNotFound
	}
	if _, exists := feeds[feedKey]; !exists {
		return ErrFeedNotFound
	}
	delete(feeds, feedKey)
	return writeConfig(configPath, config)
}

// Callers hold the service lock across the entire read-modify-write operation.
func readConfig(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func writeConfig(path string, config map[string]interface{}) error {
	content, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	// Write in place: deployments bind-mount this individual file into both containers.
	return os.WriteFile(path, content, 0644)
}
