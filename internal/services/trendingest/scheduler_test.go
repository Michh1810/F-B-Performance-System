package trendingest

import (
	"testing"
	"time"
)

func TestNextOccurrence(t *testing.T) {
	mustParse := func(s string) time.Time {
		tm, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		return tm
	}

	tests := []struct {
		name    string
		from    time.Time
		weekday time.Weekday
		hour    int
		want    time.Time
	}{
		{
			name:    "earlier in the same target day",
			from:    mustParse("2026-08-02T01:00:00Z"), // a Sunday
			weekday: time.Sunday,
			hour:    3,
			want:    mustParse("2026-08-02T03:00:00Z"),
		},
		{
			name:    "target hour already passed today, rolls to next week",
			from:    mustParse("2026-08-02T05:00:00Z"), // a Sunday, past 03:00
			weekday: time.Sunday,
			hour:    3,
			want:    mustParse("2026-08-09T03:00:00Z"),
		},
		{
			name:    "mid-week, advances to the upcoming target weekday",
			from:    mustParse("2026-08-05T12:00:00Z"), // a Wednesday
			weekday: time.Sunday,
			hour:    3,
			want:    mustParse("2026-08-09T03:00:00Z"),
		},
		{
			name:    "exactly at the target instant rolls to next week, not itself",
			from:    mustParse("2026-08-02T03:00:00Z"),
			weekday: time.Sunday,
			hour:    3,
			want:    mustParse("2026-08-09T03:00:00Z"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextOccurrence(tt.from, tt.weekday, tt.hour)
			if !got.Equal(tt.want) {
				t.Errorf("nextOccurrence(%s, %s, %d) = %s, want %s", tt.from, tt.weekday, tt.hour, got, tt.want)
			}
			if got.Weekday() != tt.weekday {
				t.Errorf("got weekday %s, want %s", got.Weekday(), tt.weekday)
			}
		})
	}
}
