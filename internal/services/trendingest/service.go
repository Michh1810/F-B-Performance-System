package trendingest

import (
	"context"
	"sync"
	"time"
)

// RunStatus is a point-in-time snapshot of the single background sweep
// this process tracks — there is deliberately only ever one "current" run
// (see Service.TriggerAsync), matching cmd/trend-ingest's own assumption
// that overlapping sweeps aren't supported.
type RunStatus struct {
	Status         string // "idle", "running", "completed", or "failed"
	Hashtags       []string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	VideosUpserted int
	Error          string
}

// Service pairs an Ingester with in-memory state tracking of its most
// recent/current run, so an HTTP handler can trigger a sweep without
// blocking the request (a real sweep takes minutes — see
// cmd/trend-ingest's doc comment) and a separate handler can report
// progress for the frontend to poll.
//
// State is process-local, not persisted — acceptable here since this
// mirrors cmd/trend-ingest's own single-process, no-durability design, and
// a server restart mid-sweep would abort the goroutine anyway.
type Service struct {
	ingester *Ingester

	mu      sync.Mutex
	current RunStatus
}

func NewService(ingester *Ingester) *Service {
	return &Service{ingester: ingester, current: RunStatus{Status: "idle"}}
}

// Status returns a snapshot of the current/most recent run.
func (s *Service) Status() RunStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// TriggerAsync starts a sweep of hashtags in the background and returns
// immediately. It returns false (a no-op) if a sweep is already running or
// hashtags is empty — guarding against overlapping runs the same way
// cmd/trend-ingest's README asks an external scheduler to, since here the
// "scheduler" is just whichever approve click happened first.
func (s *Service) TriggerAsync(hashtags []string) bool {
	if len(hashtags) == 0 {
		return false
	}

	s.mu.Lock()
	if s.current.Status == "running" {
		s.mu.Unlock()
		return false
	}
	now := time.Now().UTC()
	s.current = RunStatus{Status: "running", Hashtags: hashtags, StartedAt: &now}
	s.mu.Unlock()

	go func() {
		summary := s.ingester.Run(context.Background(), hashtags)
		completedAt := time.Now().UTC()

		// Every hashtag's fetch call failing outright (bad Apify token,
		// Apify down, etc.) is a real failure, not a healthy zero-result
		// run — report it as "failed" so the frontend's waiting screen
		// doesn't quietly say "done" for a sweep that scraped nothing
		// because it errored out immediately. A partial failure (some
		// hashtags succeeded) still reports "completed", matching Run's
		// existing best-effort semantics.
		status := "completed"
		errMsg := ""
		if summary.HashtagsSwept > 0 && summary.HashtagsFailed == summary.HashtagsSwept {
			status = "failed"
			if summary.LastFetchError != nil {
				errMsg = summary.LastFetchError.Error()
			}
		}

		s.mu.Lock()
		s.current.Status = status
		s.current.CompletedAt = &completedAt
		s.current.VideosUpserted = summary.VideosUpserted
		s.current.Error = errMsg
		s.mu.Unlock()
	}()

	return true
}
