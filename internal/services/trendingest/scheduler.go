package trendingest

import (
	"context"
	"log"
	"time"
)

// HashtagSource resolves which hashtags a scheduled sweep should use.
type HashtagSource interface {
	// LatestApproved returns the hashtags a human most recently approved,
	// or nil if none has ever been approved.
	LatestApproved(ctx context.Context) ([]string, error)
}

// Scheduler triggers a Service sweep once a week on a fixed UTC
// weekday/hour, using whatever hashtags are currently approved (falling
// back to a configured default list) — an addition on top of
// cmd/trend-ingest itself, which remains a non-self-scheduling CLI (see its
// doc comment); this is the API server choosing to run that same logic on
// a timer so the corpus stays fresh without anyone remembering to click
// Approve or run the binary by hand.
type Scheduler struct {
	service         *Service
	hashtags        HashtagSource
	defaultHashtags []string
	weekday         time.Weekday
	hour            int // UTC hour of day to fire, 0-23
}

func NewScheduler(service *Service, hashtags HashtagSource, defaultHashtags []string, weekday time.Weekday, hour int) *Scheduler {
	return &Scheduler{service: service, hashtags: hashtags, defaultHashtags: defaultHashtags, weekday: weekday, hour: hour}
}

// Start launches the scheduling loop in a background goroutine and returns
// immediately. The loop exits when ctx is canceled.
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	for {
		next := nextOccurrence(time.Now().UTC(), s.weekday, s.hour)
		log.Printf("trendingest: scheduler: next weekly sweep at %s", next.Format(time.RFC3339))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-timer.C:
			s.trigger(ctx)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *Scheduler) trigger(ctx context.Context) {
	hashtags := s.defaultHashtags
	if approved, err := s.hashtags.LatestApproved(ctx); err != nil {
		log.Printf("trendingest: scheduler: load approved hashtags: %v (falling back to configured default)", err)
	} else if len(approved) > 0 {
		hashtags = approved
	}

	if s.service.TriggerAsync(hashtags) {
		log.Printf("trendingest: scheduler: triggered weekly sweep of %d hashtags", len(hashtags))
	} else {
		log.Print("trendingest: scheduler: a sweep was already running, skipping this week's trigger")
	}
}

// nextOccurrence returns the next instant at or after `from` that falls on
// `weekday` at `hour`:00:00 UTC.
func nextOccurrence(from time.Time, weekday time.Weekday, hour int) time.Time {
	candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, 0, 0, 0, time.UTC)
	for candidate.Weekday() != weekday || !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}
