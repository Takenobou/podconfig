package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// YouTubeClient retrieves channel metadata independently of config storage.
type YouTubeClient struct {
	client http.Client
}

func NewYouTubeClient() *YouTubeClient {
	return &YouTubeClient{client: http.Client{Timeout: 15 * time.Second}}
}

// FetchChannelInfo retrieves channel info from the given YouTube URL.
func (c *YouTubeClient) FetchChannelInfo(ctx context.Context, youtubeURL string) (*NewFeedInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, youtubeURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
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
