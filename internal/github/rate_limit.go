package github

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Remaining int
	Limit     int
	Reset     time.Time
	Resource  string
}

type Limiter struct {
	mu        sync.Mutex
	remaining int
	limit     int
	reset     time.Time
	resource  string
}

func NewLimiter() *Limiter {
	return &Limiter{remaining: -1}
}

func (l *Limiter) Observe(h http.Header) {
	if h == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if v := h.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			l.remaining = n
		}
	}
	if v := h.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			l.limit = n
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			l.reset = time.Unix(n, 0)
		}
	}
	if v := h.Get("X-RateLimit-Resource"); v != "" {
		l.resource = v
	}
}

func (l *Limiter) Remaining() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.remaining
}

func (l *Limiter) Snapshot() Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return Snapshot{
		Remaining: l.remaining,
		Limit:     l.limit,
		Reset:     l.reset,
		Resource:  l.resource,
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	remaining := l.remaining
	reset := l.reset
	l.mu.Unlock()
	if remaining != 0 {
		return nil
	}
	wait := time.Until(reset) + time.Second
	if wait <= 0 {
		return nil
	}
	if wait > 5*time.Minute {
		wait = 5 * time.Minute
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func RetryAfter(h http.Header) time.Duration {
	raw := strings.TrimSpace(h.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if n, err := strconv.Atoi(raw); err == nil {
		d := time.Duration(n) * time.Second
		if d > 5*time.Minute {
			return 5 * time.Minute
		}
		if d < 0 {
			return 0
		}
		return d
	}
	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d > 5*time.Minute {
			return 5 * time.Minute
		}
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func resetWait(h http.Header) time.Duration {
	raw := strings.TrimSpace(h.Get("X-RateLimit-Reset"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	d := time.Until(time.Unix(n, 0)) + time.Second
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	if d < 0 {
		return 0
	}
	return d
}

func isRateLimit(resp *http.Response, body []byte) bool {
	if resp == nil {
		return false
	}
	if RetryAfter(resp.Header) > 0 {
		return true
	}
	if headerInt(resp.Header, "X-RateLimit-Remaining") == 0 && resp.Header.Get("X-RateLimit-Remaining") != "" {
		return true
	}
	msg := strings.ToLower(string(body))
	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "secondary rate") {
		return true
	}
	return strings.EqualFold(resp.Header.Get("X-RateLimit-Remaining"), "0")
}
