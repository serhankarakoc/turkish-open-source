package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL    = "https://api.github.com"
	defaultAPIVersion = "2026-03-10"
	defaultUserAgent  = "turkish-open-source-scanner/1.0 (+https://github.com/serhankarakoc/turkish-open-source)"
)

type Logger interface {
	Printf(format string, v ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

type Client struct {
	baseURL    *url.URL
	token      string
	apiVersion string
	userAgent  string
	http       *http.Client
	maxRetries int
	backoff    time.Duration
	maxBackoff time.Duration
	limiter    *Limiter
	cache      *etagCache
	log        Logger

	mu       sync.Mutex
	requests int
}

type Options struct {
	BaseURL        string
	APIVersion     string
	Token          string
	UserAgent      string
	Timeout        time.Duration
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Logger         Logger
	HTTPClient     *http.Client
}

func NewClient(opts Options) (*Client, error) {
	base := opts.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return nil, fmt.Errorf("github api url: %w", err)
	}
	version := opts.APIVersion
	if version == "" {
		version = defaultAPIVersion
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	retries := opts.MaxRetries
	if retries < 0 {
		retries = 5
	}
	backoff := opts.InitialBackoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	maxBackoff := opts.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 60 * time.Second
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	log := opts.Logger
	if log == nil {
		log = nopLogger{}
	}
	return &Client{
		baseURL:    u,
		token:      strings.TrimSpace(opts.Token),
		apiVersion: version,
		userAgent:  firstNonEmpty(opts.UserAgent, defaultUserAgent),
		http:       httpClient,
		maxRetries: retries,
		backoff:    backoff,
		maxBackoff: maxBackoff,
		limiter:    NewLimiter(),
		cache:      newETagCache(),
		log:        log,
	}, nil
}

func (c *Client) HasToken() bool {
	return c.token != ""
}

func (c *Client) RequestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

func (c *Client) RateLimit() Snapshot {
	return c.limiter.Snapshot()
}

func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, dest any) error {
	_, err := c.doJSON(ctx, http.MethodGet, path, query, dest, true)
	return err
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, dest any, useCache bool) (int, error) {
	body, status, err := c.do(ctx, method, path, query, useCache)
	if err != nil {
		return status, err
	}
	if dest == nil || status == http.StatusNoContent {
		return status, nil
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return status, nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return status, fmt.Errorf("decode %s: %w", path, err)
	}
	return status, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, useCache bool) ([]byte, int, error) {
	u, err := c.resolve(path, query)
	if err != nil {
		return nil, 0, err
	}
	key := method + " " + u.String()

	var lastErr error
	backoff := c.backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, 0, err
		}

		req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
		if err != nil {
			return nil, 0, err
		}
		c.applyHeaders(req)
		if useCache && method == http.MethodGet {
			if etag := c.cache.etag(key); etag != "" {
				req.Header.Set("If-None-Match", etag)
			}
		}

		c.mu.Lock()
		c.requests++
		c.mu.Unlock()

		c.log.Printf("github %s %s (attempt %d)", method, redactURL(u), attempt+1)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt == c.maxRetries || !retryableNetErr(err) {
				return nil, 0, err
			}
			if err := sleepBackoff(ctx, backoff); err != nil {
				return nil, 0, err
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt == c.maxRetries {
				return nil, resp.StatusCode, readErr
			}
			if err := sleepBackoff(ctx, backoff); err != nil {
				return nil, 0, err
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}

		c.limiter.Observe(resp.Header)
		c.log.Printf("github %s %s -> %d remaining=%d", method, redactURL(u), resp.StatusCode, c.limiter.Remaining())

		switch resp.StatusCode {
		case http.StatusOK:
			if etag := resp.Header.Get("ETag"); etag != "" && useCache {
				c.cache.store(key, etag, body)
			}
			return body, resp.StatusCode, nil
		case http.StatusNotModified:
			cached, ok := c.cache.body(key)
			if !ok {
				return nil, resp.StatusCode, fmt.Errorf("github 304 without cached body for %s", path)
			}
			return cached, http.StatusOK, nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
			return body, resp.StatusCode, APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512)}
		case http.StatusForbidden:
			if isRateLimit(resp, body) {
				lastErr = APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512), RateLimited: true}
				if attempt == c.maxRetries {
					return body, resp.StatusCode, lastErr
				}
				if err := c.waitRateLimit(ctx, resp); err != nil {
					return body, resp.StatusCode, err
				}
				backoff = nextBackoff(backoff, c.maxBackoff)
				continue
			}
			return body, resp.StatusCode, APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512)}
		case http.StatusTooManyRequests:
			lastErr = APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512), RateLimited: true}
			if attempt == c.maxRetries {
				return body, resp.StatusCode, lastErr
			}
			if err := c.waitRateLimit(ctx, resp); err != nil {
				return body, resp.StatusCode, err
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			lastErr = APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512)}
			if attempt == c.maxRetries {
				return body, resp.StatusCode, lastErr
			}
			if err := sleepBackoff(ctx, backoff); err != nil {
				return body, resp.StatusCode, err
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		default:
			if resp.StatusCode >= 400 {
				return body, resp.StatusCode, APIError{Status: resp.StatusCode, Path: path, Body: truncate(body, 512)}
			}
			return body, resp.StatusCode, nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("github: exhausted retries")
	}
	return nil, 0, lastErr
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", c.apiVersion)
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *Client) resolve(path string, query url.Values) (*url.URL, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		if query != nil {
			q := u.Query()
			for k, vs := range query {
				for _, v := range vs {
					q.Set(k, v)
				}
			}
			u.RawQuery = q.Encode()
		}
		return u, nil
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u := c.baseURL.ResolveReference(rel)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u, nil
}

func (c *Client) waitRateLimit(ctx context.Context, resp *http.Response) error {
	if d := RetryAfter(resp.Header); d > 0 {
		c.log.Printf("rate limit: waiting Retry-After %s", d)
		return sleepBackoff(ctx, d)
	}
	if remaining := headerInt(resp.Header, "X-RateLimit-Remaining"); remaining == 0 {
		if reset := resetWait(resp.Header); reset > 0 {
			c.log.Printf("rate limit: waiting until reset (%s)", reset)
			return sleepBackoff(ctx, reset)
		}
	}
	c.log.Printf("rate limit: exponential backoff")
	return sleepBackoff(ctx, c.backoff)
}

type APIError struct {
	Status      int
	Path        string
	Body        string
	RateLimited bool
}

func (e APIError) Error() string {
	msg := fmt.Sprintf("github API %s: HTTP %d", e.Path, e.Status)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

func IsNotFound(err error) bool {
	var api APIError
	return errors.As(err, &api) && api.Status == http.StatusNotFound
}

func IsRateLimited(err error) bool {
	var api APIError
	return errors.As(err, &api) && api.RateLimited
}

func retryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func sleepBackoff(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		d = 200 * time.Millisecond
	}
	if d > 5*time.Minute {
		d = 5 * time.Minute
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	cp := *u
	cp.User = nil
	return cp.String()
}

func headerInt(h http.Header, key string) int {
	n, _ := strconv.Atoi(h.Get(key))
	return n
}

type etagCache struct {
	mu      sync.Mutex
	entries map[string]etagEntry
}

type etagEntry struct {
	etag string
	body []byte
}

func newETagCache() *etagCache {
	return &etagCache{entries: map[string]etagEntry{}}
}

func (c *etagCache) etag(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entries[key].etag
}

func (c *etagCache) body(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	return e.body, ok
}

func (c *etagCache) store(key, etag string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]byte, len(body))
	copy(cp, body)
	c.entries[key] = etagEntry{etag: etag, body: cp}
}
