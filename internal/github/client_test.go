package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClient(Options{
		BaseURL:        srv.URL,
		APIVersion:     "2026-03-10",
		Token:          "test-token",
		Timeout:        2 * time.Second,
		MaxRetries:     3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHeadersAndJSONDecode(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Accept=%s", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header presence")
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
			t.Errorf("version=%s", r.Header.Get("X-GitHub-Api-Version"))
		}
		if strings.Contains(strings.ToLower(r.Header.Get("Authorization")), "log") {
			t.Fatal("should not happen")
		}
		_ = json.NewEncoder(w).Encode(User{Login: "octocat", Type: "User"})
	})
	u, err := c.GetUser(context.Background(), "octocat")
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "octocat" {
		t.Fatalf("%+v", u)
	}
}

func TestDoesNotRetry401(t *testing.T) {
	var hits atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, `{"message":"bad credentials"}`, http.StatusUnauthorized)
	})
	err := c.GetJSON(context.Background(), "/user", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Fatalf("retries=%d", hits.Load())
	}
}

func TestRetry500ThenSuccess(t *testing.T) {
	var hits atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			http.Error(w, "nope", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"login":"ok"}`)
	})
	var u User
	if err := c.GetJSON(context.Background(), "/users/ok", nil, &u); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestRetryAfter429(t *testing.T) {
	var hits atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("X-RateLimit-Remaining", "1")
			http.Error(w, `{"message":"rate limit"}`, http.StatusTooManyRequests)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", "42")
		_, _ = io.WriteString(w, `{"login":"ok"}`)
	})
	var u User
	if err := c.GetJSON(context.Background(), "/users/ok", nil, &u); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
	if c.RateLimit().Remaining != 42 {
		t.Fatalf("remaining=%d", c.RateLimit().Remaining)
	}
}

func TestSecondaryRateLimit403(t *testing.T) {
	var hits atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, `{"message":"You have exceeded a secondary rate limit"}`, http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, `{"login":"ok"}`)
	})
	var u User
	if err := c.GetJSON(context.Background(), "/users/ok", nil, &u); err != nil {
		t.Fatal(err)
	}
}

func TestNotFound(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	_, err := c.GetRepository(context.Background(), "a", "b")
	if !IsNotFound(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestConditionalGet304(t *testing.T) {
	var hits atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("ETag", `"abc"`)
			_, _ = io.WriteString(w, `{"login":"cached"}`)
			return
		}
		if r.Header.Get("If-None-Match") != `"abc"` {
			t.Errorf("missing if-none-match")
		}
		w.WriteHeader(http.StatusNotModified)
	})
	var u User
	if err := c.GetJSON(context.Background(), "/users/cached", nil, &u); err != nil {
		t.Fatal(err)
	}
	u = User{}
	if err := c.GetJSON(context.Background(), "/users/cached", nil, &u); err != nil {
		t.Fatal(err)
	}
	if u.Login != "cached" {
		t.Fatalf("%+v", u)
	}
}

func TestSearchPagination(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("X-RateLimit-Remaining", "20")
		switch page {
		case "", "1":
			_, _ = io.WriteString(w, `{"total_count":2,"incomplete_results":false,"items":[{"id":1,"name":"one","full_name":"o/one"}]}`)
		case "2":
			_, _ = io.WriteString(w, `{"total_count":2,"incomplete_results":false,"items":[{"id":2,"name":"two","full_name":"o/two"}]}`)
		default:
			t.Errorf("unexpected page %s", page)
			http.Error(w, "bad page", 500)
		}
	})
	repos, err := c.SearchRepositories(context.Background(), "topic:turkey", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos", len(repos))
	}
}

func TestListOwnerReposUsesOrgPath(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/acme/repos" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":1,"name":"repo","full_name":"acme/repo"}]`)
	})
	repos, err := c.ListOwnerRepos(context.Background(), "acme", "Organization", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].FullName != "acme/repo" {
		t.Fatalf("%+v", repos)
	}
}

func TestGetOrganizationAndLanguages(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/primefaces":
			_ = json.NewEncoder(w).Encode(User{Login: "primefaces", Type: "Organization", Location: "Turkey"})
		case "/repos/primefaces/primeng/languages":
			_ = json.NewEncoder(w).Encode(map[string]int{"TypeScript": 900, "CSS": 10})
		default:
			http.NotFound(w, r)
		}
	})
	org, err := c.GetOrganization(context.Background(), "primefaces")
	if err != nil {
		t.Fatal(err)
	}
	if org.Login != "primefaces" || org.Location != "Turkey" {
		t.Fatalf("%+v", org)
	}
	langs, err := c.GetLanguages(context.Background(), "primefaces", "primeng")
	if err != nil {
		t.Fatal(err)
	}
	if PrimaryLanguage(langs) != "TypeScript" {
		t.Fatalf("%v", langs)
	}
}

func TestParseLinkNext(t *testing.T) {
	next := ParseLinkNext(`<https://api.github.com/search/repositories?q=foo&page=2>; rel="next", <https://api.github.com/search/repositories?q=foo&page=10>; rel="last"`)
	if !strings.Contains(next, "page=2") {
		t.Fatalf("next=%s", next)
	}
	if ParseLinkNext("") != "" {
		t.Fatal("empty header")
	}
}

func TestTokenNotLogged(t *testing.T) {
	var buf strings.Builder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"login":"x"}`)
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Options{
		BaseURL: srv.URL,
		Token:   "super-secret-token-value",
		Logger:  loggerFunc(func(format string, v ...any) { buf.WriteString(strings.TrimSpace(format) + "\n") }),
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = c.GetJSON(context.Background(), "/users/x", nil, &User{})
	if strings.Contains(buf.String(), "super-secret-token-value") {
		t.Fatal("token leaked into logs")
	}
	if strings.Contains(strings.ToLower(buf.String()), "authorization") {
		t.Fatal("authorization header must not be logged")
	}
}

type loggerFunc func(string, ...any)

func (f loggerFunc) Printf(format string, v ...any) { f(format, v...) }
