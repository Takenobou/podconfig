package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func testHTMXHandler(t *testing.T) *Handler {
	t.Helper()
	client := NewYouTubeClient()
	client.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<link rel="canonical" href="https://www.youtube.com/channel/UC123"><meta property="og:title" content="Demo &amp; Friends">`))}, nil
	})
	return &Handler{
		PodsyncConfigPath:   testConfig(t, "[server]\nhostname='https://podsync.test'\n[feeds]\n"),
		DockerContainerName: "podsync", FeedService: &FeedService{}, YouTubeClient: client,
	}
}

func postHTMX(t *testing.T, handler http.HandlerFunc, path string, form url.Values, status int) *goquery.Document {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != status {
		t.Fatalf("%s: got HTTP %d, want %d: %s", path, w.Code, status, w.Body.String())
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected HTML, got %s", w.Header().Get("Content-Type"))
	}
	if strings.Contains(w.Body.String(), "<!DOCTYPE") {
		t.Fatal("HTMX response contains a full page")
	}
	doc, err := goquery.NewDocumentFromReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestHTMXFeedWorkflow(t *testing.T) {
	h := testHTMXHandler(t)
	form := url.Values{"youtubeUrl": {"https://www.youtube.com/@demo"}}
	added := postHTMX(t, h.AddFeedHandler, "/add", form, http.StatusOK)
	if added.Find("#feedListWrapper[hx-swap-oob] .feed-item").Length() != 1 {
		t.Fatal("added feed missing from swap")
	}
	if !strings.Contains(added.Find("#changelogWrapper[hx-swap-oob]").Text(), "Added feed") {
		t.Fatal("pending change missing")
	}
	if !strings.Contains(added.Find(".message").First().Text(), "Demo & Friends") {
		t.Fatal("channel title missing")
	}
	before, _ := os.ReadFile(h.PodsyncConfigPath)
	duplicate := postHTMX(t, h.AddFeedHandler, "/add", form, http.StatusConflict)
	if duplicate.Find("[hx-swap-oob]").Length() != 0 || !strings.Contains(duplicate.Text(), "already exists") {
		t.Fatal("duplicate should update only the error message")
	}
	after, _ := os.ReadFile(h.PodsyncConfigPath)
	if string(before) != string(after) {
		t.Fatal("duplicate changed config")
	}
	edited := postHTMX(t, h.ModifyFeedHandler, "/modify", url.Values{"feedKey": {"demofriends"}, "max_age": {"30"}}, http.StatusOK)
	if edited.Find("input[name=max_age]").AttrOr("value", "") != "30" {
		t.Fatal("updated setting missing from swap")
	}
	invalid := postHTMX(t, h.ModifyFeedHandler, "/modify", url.Values{"feedKey": {"demofriends"}, "max_age": {"invalid"}}, http.StatusBadRequest)
	if invalid.Find("[hx-swap-oob]").Length() != 0 {
		t.Fatal("invalid update should leave forms intact")
	}
	removed := postHTMX(t, h.RemoveFeedHandler, "/remove", url.Values{"feedKey": {"demofriends"}}, http.StatusOK)
	if removed.Find(".feed-item").Length() != 0 || !strings.Contains(removed.Find("#feedListWrapper").Text(), "No feeds found") {
		t.Fatal("removed feed still displayed")
	}
	if len(h.getChanges()) != 3 {
		t.Fatalf("unexpected changelog: %v", h.getChanges())
	}
	postHTMX(t, h.RemoveFeedHandler, "/remove", url.Values{"feedKey": {"missing"}}, http.StatusNotFound)
}

func TestPlainFormRedirectsAfterMutation(t *testing.T) {
	h := testHTMXHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader("youtubeUrl=https://www.youtube.com/@demo"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.AddFeedHandler(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("got HTTP %d: %s", w.Code, w.Body.String())
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil || location.Path != "/" || !strings.Contains(location.Query().Get("message"), "added successfully") {
		t.Fatalf("unexpected redirect: %s", w.Header().Get("Location"))
	}
}

func TestHTMXMessagesEscapeHTML(t *testing.T) {
	h := testHTMXHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/add", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	h.renderError(w, req, `<img src=x onerror="alert(1)">`, http.StatusBadRequest)
	if strings.Contains(w.Body.String(), "<img") || !strings.Contains(w.Body.String(), "&lt;img") {
		t.Fatalf("message was not escaped: %s", w.Body.String())
	}
}

func TestIndexRendersInitialState(t *testing.T) {
	h := testHTMXHandler(t)
	h.addChange("Existing pending change")
	w := httptest.NewRecorder()
	h.Index(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP %d", w.Code)
	}
	doc, err := goquery.NewDocumentFromReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Find("#changelogWrapper").Text(), "Existing pending change") {
		t.Fatal("initial changelog missing")
	}
	if doc.Find("script[src='/static/vendor/htmx-2.0.11.min.js']").Length() != 1 {
		t.Fatal("local HTMX script missing")
	}
	if doc.Find("[hx-get]").Length() != 0 {
		t.Fatal("initial state should not need follow-up requests")
	}
	if doc.Find("#addFeedForm [name=update_period]").Length() != 1 {
		t.Fatal("advanced fields must submit named values")
	}
}

func TestHTMXReload(t *testing.T) {
	for _, success := range []bool{true, false} {
		name := "failure"
		if success {
			name = "success"
		}
		t.Run(name, func(t *testing.T) {
			var restarts atomic.Int32
			docker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1.48/containers/podsync/restart" {
					http.Error(w, "unexpected Docker request", http.StatusNotFound)
					return
				}
				restarts.Add(1)
				if !success {
					http.Error(w, "restart failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer docker.Close()
			t.Setenv("DOCKER_HOST", strings.Replace(docker.URL, "http://", "tcp://", 1))
			t.Setenv("DOCKER_API_VERSION", "1.48")
			h := testHTMXHandler(t)
			h.addChange("Added a feed")
			status := http.StatusInternalServerError
			if success {
				status = http.StatusOK
			}
			doc := postHTMX(t, h.ReloadHandler, "/reload", nil, status)
			if restarts.Load() != 1 {
				t.Fatalf("got %d restarts", restarts.Load())
			}
			if success {
				if len(h.getChanges()) != 0 || !strings.Contains(doc.Find("#changelogWrapper").Text(), "No pending changes") {
					t.Fatal("successful reload did not clear pending changes")
				}
			} else {
				if len(h.getChanges()) != 1 || doc.Find("[hx-swap-oob]").Length() != 0 {
					t.Fatal("failed reload should preserve pending changes and forms")
				}
			}
		})
	}
}
