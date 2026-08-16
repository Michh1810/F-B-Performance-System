package tiktok

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer serves both the run-sync endpoint and a comments dataset
// endpoint from the same httptest.Server, since attachComments follows
// whatever URL the run-sync response gives it for commentsDatasetUrl.
func newTestServer(t *testing.T, videosJSON string, commentsJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/acts/test-actor/run-sync-get-dataset-items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, videosJSON)
	})
	mux.HandleFunc("/comments-dataset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, commentsJSON)
	})
	return httptest.NewServer(mux)
}

func TestFetchByHashtags_AttachesCommentsByVideoURL(t *testing.T) {
	videosJSON := `[
		{"id": "1", "text": "video one", "webVideoUrl": "https://www.tiktok.com/@a/video/1", "commentsDatasetUrl": "%[1]s/comments-dataset"},
		{"id": "2", "text": "video two", "webVideoUrl": "https://www.tiktok.com/@b/video/2", "commentsDatasetUrl": "%[1]s/comments-dataset"}
	]`
	commentsJSON := `[
		{"videoWebUrl": "https://www.tiktok.com/@a/video/1", "cid": "c1", "text": "comment on video one", "diggCount": 5},
		{"videoWebUrl": "https://www.tiktok.com/@b/video/2", "cid": "c2", "text": "comment on video two", "diggCount": 9},
		{"videoWebUrl": "https://www.tiktok.com/@b/video/2", "cid": "c3", "text": "another comment on video two", "diggCount": 2}
	]`

	// The videos response needs to embed the server's own URL for
	// commentsDatasetUrl, so build the server first with a placeholder
	// handler, then format the fixture with its real address.
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/acts/test-actor/run-sync-get-dataset-items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, videosJSON, srv.URL)
	})
	mux.HandleFunc("/comments-dataset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, commentsJSON)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient("test-token", "test-actor", WithBaseURL(srv.URL))
	videos, err := client.FetchByHashtags(context.Background(), []string{"test"}, 10)
	if err != nil {
		t.Fatalf("FetchByHashtags: %v", err)
	}
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}

	var v1, v2 Video
	for _, v := range videos {
		switch v.ID {
		case "1":
			v1 = v
		case "2":
			v2 = v
		}
	}

	if len(v1.Comments) != 1 || v1.Comments[0].Text != "comment on video one" {
		t.Fatalf("video 1 comments = %+v, want 1 comment matching by videoWebUrl", v1.Comments)
	}
	if len(v2.Comments) != 2 {
		t.Fatalf("video 2 comments = %+v, want 2 comments matching by videoWebUrl", v2.Comments)
	}
}

func TestFetchByHashtags_NoCommentsDatasetURL_LeavesCommentsEmpty(t *testing.T) {
	videosJSON := `[{"id": "1", "text": "video one", "webVideoUrl": "https://www.tiktok.com/@a/video/1"}]`
	srv := newTestServer(t, videosJSON, `[]`)
	defer srv.Close()

	client := NewClient("test-token", "test-actor", WithBaseURL(srv.URL))
	videos, err := client.FetchByHashtags(context.Background(), []string{"test"}, 10)
	if err != nil {
		t.Fatalf("FetchByHashtags: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	if videos[0].Comments != nil {
		t.Fatalf("expected nil Comments when no commentsDatasetUrl is present, got %v", videos[0].Comments)
	}
}

func TestFetchByHashtags_CommentsFetchFailureDoesNotFailWholeCall(t *testing.T) {
	videosJSON := `[{"id": "1", "text": "video one", "webVideoUrl": "https://www.tiktok.com/@a/video/1", "commentsDatasetUrl": "%[1]s/missing-dataset"}]`
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/acts/test-actor/run-sync-get-dataset-items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, videosJSON, srv.URL)
	})
	// Deliberately no handler for /missing-dataset -- returns 404.
	srv = httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient("test-token", "test-actor", WithBaseURL(srv.URL))
	videos, err := client.FetchByHashtags(context.Background(), []string{"test"}, 10)
	if err != nil {
		t.Fatalf("FetchByHashtags() error = %v, want nil (comments fetch failure should be best-effort)", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	if videos[0].Comments != nil {
		t.Fatalf("expected nil Comments when the comments fetch fails, got %v", videos[0].Comments)
	}
}
