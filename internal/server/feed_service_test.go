package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConcurrentFeedChanges(t *testing.T) {
	path := testConfig(t, "[server]\nhostname='https://podsync.test'\n[feeds]\n")
	service := &FeedService{}
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("old%d", i)
		if err := service.AppendFeedToConfig(path, &NewFeedInfo{FeedKey: key}, FeedUpdate{}); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			<-start
			if err := service.RemoveFeed(path, fmt.Sprintf("old%d", i)); err != nil {
				t.Error(err)
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			<-start
			if err := service.AppendFeedToConfig(path, &NewFeedInfo{FeedKey: fmt.Sprintf("new%d", i)}, FeedUpdate{}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	feeds, err := service.GetFeedList(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 20 {
		t.Fatalf("got %d feeds, want 20", len(feeds))
	}
	for _, feed := range feeds {
		if len(feed.Key) < 3 || feed.Key[:3] != "new" {
			t.Errorf("removed feed remains: %s", feed.Key)
		}
		if feed.XMLURL != "https://podsync.test/"+feed.Key+".xml" {
			t.Errorf("server settings changed: %s", feed.XMLURL)
		}
	}
}

func TestRemoveMissingFeedPreservesConfig(t *testing.T) {
	path := testConfig(t, "[feeds.demo]\nurl='original'\n")
	before, _ := os.ReadFile(path)
	if err := (&FeedService{}).RemoveFeed(path, "missing"); !errors.Is(err, ErrFeedNotFound) {
		t.Fatalf("got %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("missing removal changed config")
	}
}

func TestModifyPreservesOtherSettings(t *testing.T) {
	path := testConfig(t, `[server]
hostname='https://podsync.test'
[feeds.demo]
url='original'
quality='low'
[feeds.demo.filters]
max_age=90
min_duration=120
[feeds.demo.clean]
keep_last=20
custom_setting='preserve'
[feeds.demo.custom]
title='Demo'
`)
	maxAge, keepLast := 30, 0
	if err := (&FeedService{}).ModifyFeed(path, "demo", FeedUpdate{MaxAge: &maxAge, CleanKeepLast: &keepLast}); err != nil {
		t.Fatal(err)
	}
	config, err := readConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	feed := config["feeds"].(map[string]interface{})["demo"].(map[string]interface{})
	filters := feed["filters"].(map[string]interface{})
	clean := feed["clean"].(map[string]interface{})
	if filters["min_duration"] != int64(120) || filters["max_age"] != int64(30) {
		t.Fatalf("filters: %#v", filters)
	}
	if clean["custom_setting"] != "preserve" || clean["keep_last"] != int64(0) {
		t.Fatalf("clean: %#v", clean)
	}
	if feed["quality"] != "low" || feed["url"] != "original" || feed["custom"].(map[string]interface{})["title"] != "Demo" {
		t.Fatalf("other settings changed: %#v", feed)
	}
}

func TestAddUsesServiceDefaults(t *testing.T) {
	path := testConfig(t, "[feeds]\n")
	service := &FeedService{}
	if err := service.AppendFeedToConfig(path, &NewFeedInfo{FeedKey: "demo"}, FeedUpdate{}); err != nil {
		t.Fatal(err)
	}
	feeds, err := service.GetFeedList(path)
	if err != nil {
		t.Fatal(err)
	}
	feed := feeds[0]
	if feed.UpdatePeriod != "1h" || feed.Format != "video" || feed.CleanKeepLast != "20" || feed.MaxAge != "90" {
		t.Fatalf("unexpected defaults: %#v", feed)
	}
}
