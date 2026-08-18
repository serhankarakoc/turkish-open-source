package validation

import (
	"testing"
	"time"
)

func TestActivityScoreRecentPush(t *testing.T) {
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	got := ActivityScore(ActivityInput{
		PushedAt: now.Add(-10 * 24 * time.Hour),
		Now:      now,
	})
	if got < 40 {
		t.Fatalf("recent push should score high, got %d", got)
	}
	old := ActivityScore(ActivityInput{
		PushedAt: now.Add(-800 * 24 * time.Hour),
		Now:      now,
	})
	if old >= got {
		t.Fatalf("stale project should score lower: recent=%d old=%d", got, old)
	}
}

func TestActivityScoreReleaseAndContributors(t *testing.T) {
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	got := ActivityScore(ActivityInput{
		PushedAt:         now.Add(-40 * 24 * time.Hour),
		HasRecentRelease: true,
		ReleaseAt:        now.Add(-20 * 24 * time.Hour),
		ContributorCount: 6,
		OpenIssues:       3,
		HasIssueActivity: true,
		Now:              now,
	})
	if got < 50 {
		t.Fatalf("expected a substantial activity score, got %d", got)
	}
	if got > 100 {
		t.Fatalf("score must be normalized to 100, got %d", got)
	}
}

func TestIsActive(t *testing.T) {
	now := time.Now().UTC()
	if IsActive(true, now, now, 365) {
		t.Fatal("archived repos are not active")
	}
	if !IsActive(false, now.Add(-10*24*time.Hour), now, 365) {
		t.Fatal("recent push should be active")
	}
}
