package server

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestModifyRejectsInvalidSettings(t *testing.T) {
	for _, form := range []string{"max_age=nope", "max_age=-1", "clean_keep_last=-1", "format=other", "update_period=invalid", "update_period=0s"} {
		t.Run(form, func(t *testing.T) {
			path := testConfig(t, "[feeds.demo]\nurl='original'\n")
			before, _ := os.ReadFile(path)
			h := &Handler{PodsyncConfigPath: path, FeedService: &FeedService{}}
			req := httptest.NewRequest("POST", "/modify", strings.NewReader("feedKey=demo&"+form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			h.ModifyFeedHandler(w, req)
			if w.Code != 400 {
				t.Fatalf("got HTTP %d: %s", w.Code, w.Body.String())
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("invalid request changed config")
			}
			if len(h.getChanges()) != 0 {
				t.Fatal("invalid request recorded a change")
			}
		})
	}
}
