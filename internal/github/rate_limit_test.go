package github

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryAfterSeconds(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "7")
	if RetryAfter(h) != 7*time.Second {
		t.Fatalf("got %s", RetryAfter(h))
	}
}

func TestLimiterObserve(t *testing.T) {
	l := NewLimiter()
	h := http.Header{}
	h.Set("X-RateLimit-Remaining", "12")
	h.Set("X-RateLimit-Limit", "30")
	h.Set("X-RateLimit-Resource", "search")
	h.Set("X-RateLimit-Reset", "2000000000")
	l.Observe(h)
	snap := l.Snapshot()
	if snap.Remaining != 12 || snap.Limit != 30 || snap.Resource != "search" {
		t.Fatalf("%+v", snap)
	}
}

func TestIsRateLimit(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	if !isRateLimit(resp, []byte(`{"message":"API rate limit exceeded"}`)) {
		t.Fatal("expected rate limit")
	}
	resp2 := &http.Response{Header: http.Header{}}
	if isRateLimit(resp2, []byte(`{"message":"nope"}`)) {
		t.Fatal("should not treat generic 403 as rate limit")
	}
}

func TestTurkishTopics(t *testing.T) {
	got := TurkishTopics([]string{"cli", "Turkey", "made-in-turkey", "cli"})
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	if !IsTurkishTopic("türkiye") {
		t.Fatal("expected türkiye topic")
	}
}
