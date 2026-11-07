package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestFeedXMLAvailability(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		ready  bool
	}{
		{"not generated", http.StatusNotFound, "Not found", false},
		{"generated RSS", http.StatusOK, `<?xml version="1.0"?><rss version="2.0"><channel/></rss>`, true},
		{"proxy HTML response", http.StatusOK, `<html><body>Not found</body></html>`, false},
		{"upstream error", http.StatusServiceUnavailable, `<rss/>`, false},
		{"invalid XML", http.StatusOK, "invalid XML", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/demo.xml" {
					t.Errorf("unexpected probe: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer upstream.Close()
			h := &Handler{PodsyncConfigPath: testConfig(t, fmt.Sprintf("[server]\nhostname=%q\n[feeds.demo]\nurl='source'\n", upstream.URL)), FeedService: &FeedService{}}
			w := httptest.NewRecorder()
			h.FeedXMLHandler(w, httptest.NewRequest(http.MethodGet, "/feed-xml?feedKey=demo", nil))
			wantStatus := http.StatusOK
			if test.ready {
				wantStatus = stopPolling
			}
			if w.Code != wantStatus {
				t.Fatalf("got HTTP %d: %s", w.Code, w.Body.String())
			}
			doc, err := goquery.NewDocumentFromReader(w.Body)
			if err != nil {
				t.Fatal(err)
			}
			button := doc.Find("button.feed-xml")
			if button.Length() != 1 {
				t.Fatal("copy control missing")
			}
			if button.Is("[disabled]") == test.ready {
				t.Fatalf("wrong disabled state: %v", button.Is("[disabled]"))
			}
			if test.ready {
				if button.AttrOr("data-xmlurl", "") != upstream.URL+"/demo.xml" {
					t.Fatal("wrong clipboard URL")
				}
			} else if button.AttrOr("hx-on:click", "") != "" {
				t.Fatal("unavailable feed should not have a copy handler")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("availability must not be cached")
			}
		})
	}
}

func TestFeedXMLBecomesAvailable(t *testing.T) {
	var ready atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, "<rss><channel/></rss>")
	}))
	defer upstream.Close()
	h := &Handler{PodsyncConfigPath: testConfig(t, fmt.Sprintf("[server]\nhostname=%q\n[feeds.demo]\nurl='source'\n", upstream.URL)), FeedService: &FeedService{}}
	for _, available := range []bool{false, true} {
		ready.Store(available)
		w := httptest.NewRecorder()
		h.FeedXMLHandler(w, httptest.NewRequest(http.MethodGet, "/feed-xml?feedKey=demo", nil))
		if (w.Code == stopPolling) != available {
			t.Fatalf("availability did not update: HTTP %d", w.Code)
		}
	}
}

func TestFeedXMLMissingURLOrRemovedFeed(t *testing.T) {
	h := &Handler{PodsyncConfigPath: testConfig(t, "[feeds.demo]\nurl='source'\n"), FeedService: &FeedService{}}
	for _, key := range []string{"demo", "missing"} {
		w := httptest.NewRecorder()
		h.FeedXMLHandler(w, httptest.NewRequest(http.MethodGet, "/feed-xml?feedKey="+key, nil))
		if w.Code != stopPolling || !strings.Contains(w.Body.String(), "disabled") {
			t.Fatalf("unavailable URL must stop polling: HTTP %d", w.Code)
		}
	}
}

func TestFeedXMLCheckHonoursCancellation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer upstream.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if rssAvailable(ctx, upstream.URL) {
		t.Fatal("cancelled check must not enable copying")
	}
}

func TestFeedXMLStartsDisabledAndPollsIndependently(t *testing.T) {
	h := testHTMXHandler(t)
	if err := h.FeedService.AppendFeedToConfig(h.PodsyncConfigPath, &NewFeedInfo{FeedKey: "demo"}, FeedUpdate{}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.Index(w, httptest.NewRequest(http.MethodGet, "/", nil))
	doc, err := goquery.NewDocumentFromReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Find(".feed-xml[disabled]").Length() != 1 {
		t.Fatal("initial copy button must be disabled")
	}
	status := doc.Find(".feed-xml-status")
	if status.AttrOr("hx-get", "") != "/feed-xml?feedKey=demo" || status.AttrOr("hx-trigger", "") != "load, every 10s" {
		t.Fatal("availability polling missing")
	}
	if status.AttrOr("hx-sync", "") != "this:drop" {
		t.Fatal("availability checks must not block form submissions or other feeds")
	}
}
