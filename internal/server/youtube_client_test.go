package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchChannelInfo(t *testing.T) {
	for _, test := range []struct{ name, html, wantURL string }{
		{"canonical channel", `<link rel="canonical" href="https://www.youtube.com/channel/UC123">`, "https://www.youtube.com/channel/UC123"},
		{"channel handle", `<link rel="canonical" href="https://www.youtube.com/@demo"><meta itemprop="channelId" content="UC123">`, "https://www.youtube.com/channel/UC123"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := NewYouTubeClient()
			client.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.html + `<meta property="og:title" content="Demo &amp; Friends"><meta property="og:image" content="https://image.test/cover.jpg">`))}, nil
			})
			feed, err := client.FetchChannelInfo(context.Background(), "https://www.youtube.com/@demo")
			if err != nil {
				t.Fatal(err)
			}
			if feed.URL != test.wantURL || feed.ChannelName != "Demo & Friends" || feed.FeedKey != "demofriends" || feed.ProfilePicture != "https://image.test/cover.jpg" {
				t.Fatalf("unexpected metadata: %#v", feed)
			}
		})
	}
}

func TestFetchChannelInfoHonoursCancellation(t *testing.T) {
	for _, cancelRequest := range []bool{false, true} {
		t.Run(map[bool]string{false: "client timeout", true: "request cancellation"}[cancelRequest], func(t *testing.T) {
			client := NewYouTubeClient()
			if client.client.Timeout <= 0 {
				t.Fatal("default client has no timeout")
			}
			client.client.Timeout = 10 * time.Millisecond
			client.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
				<-r.Context().Done()
				return nil, r.Context().Err()
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := context.DeadlineExceeded
			if cancelRequest {
				cancel()
				want = context.Canceled
			}
			_, err := client.FetchChannelInfo(ctx, "https://www.youtube.com/@demo")
			if !errors.Is(err, want) {
				t.Fatalf("got %v, want %v", err, want)
			}
		})
	}
}
