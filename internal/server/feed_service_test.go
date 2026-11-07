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
		if err := service.AppendFeedToConfig(path, &NewFeedInfo{FeedKey: key}, "1h", "audio", 20, 90); err != nil {
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
			if err := service.AppendFeedToConfig(path, &NewFeedInfo{FeedKey: fmt.Sprintf("new%d", i)}, "1h", "audio", 20, 90); err != nil {
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
