package framework

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

func TestParseGitHubRepo(t *testing.T) {
	owner, repo, canonical, err := ParseGitHubRepo("https://github.com/kemalcr/kemal.git")
	if err != nil {
		t.Fatal(err)
	}
	if owner != "kemalcr" || repo != "kemal" || canonical != "https://github.com/kemalcr/kemal" {
		t.Fatalf("%s %s %s", owner, repo, canonical)
	}
	if _, _, _, err := ParseGitHubRepo("https://gitlab.com/foo/bar"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSeedFileHasNoHardcodedMetadata(t *testing.T) {
	root := findRepoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "config", "frameworks.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, banned := range []string{"stars:", "language:", "license:", "website:", "stargazers"} {
		if strings.Contains(strings.ToLower(text), banned) {
			t.Fatalf("seed file must not contain %s", banned)
		}
	}
	seeds, err := LoadSeeds(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) < 10 {
		t.Fatalf("expected curated seeds, got %d", len(seeds))
	}
	want := map[string]string{
		"abpframework/abp":                    "application-framework",
		"aspnetboilerplate/aspnetboilerplate": "application-framework",
		"primefaces/primefaces":               "ui-framework",
		"kemalcr/kemal":                       "web-framework",
		"znframework/znframework":             "web-framework",
		"zatrano/framework":                   "web-framework",
		"hepsiburada/voltranjs":               "micro-frontend-framework",
		"puzzle-js/puzzle-js":                 "micro-frontend-framework",
		"obss/sahi":                           "computer-vision-framework",
		"trendyol/stove":                      "testing-framework",
	}
	got := map[string]string{}
	for _, seed := range seeds {
		got[CanonicalKey(seed.GitHub)] = seed.Category
	}
	for key, category := range want {
		if got[key] != category {
			t.Fatalf("seed %s: got %q", key, got[key])
		}
	}
	for _, banned := range []string{"primefaces/primeng", "primefaces/primereact", "primefaces/primevue"} {
		if _, ok := got[banned]; ok {
			t.Fatalf("extra Prime family seed must not be present: %s", banned)
		}
	}
}

func TestScanFillsMetadataFromAPI(t *testing.T) {
	readme := base64.StdEncoding.EncodeToString([]byte("# Kemal\nA web framework from Istanbul, Turkey.\nWebsite: https://kemalcr.com\n"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/kemalcr/kemal":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":              "kemal",
				"full_name":         "kemalcr/kemal",
				"html_url":          "https://github.com/kemalcr/kemal",
				"description":       "Fast, simple web framework for Crystal",
				"language":          "Crystal",
				"homepage":          "https://kemalcr.com",
				"stargazers_count":  3901,
				"forks_count":       10,
				"open_issues_count": 2,
				"watchers_count":    3901,
				"subscribers_count": 80,
				"default_branch":    "master",
				"topics":            []string{"framework", "crystal"},
				"fork":              false,
				"archived":          false,
				"license":           map[string]string{"spdx_id": "MIT"},
				"created_at":        "2016-01-01T00:00:00Z",
				"updated_at":        "2026-08-01T00:00:00Z",
				"pushed_at":         "2026-08-01T00:00:00Z",
				"owner":             map[string]any{"login": "kemalcr", "type": "Organization"},
			})
		case "/orgs/kemalcr":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"login":    "kemalcr",
				"type":     "Organization",
				"location": "Istanbul, Turkey",
				"blog":     "https://kemalcr.com",
				"bio":      "Crystal web framework",
			})
		case "/repos/kemalcr/kemal/readme":
			_ = json.NewEncoder(w).Encode(map[string]any{"content": readme, "encoding": "base64"})
		case "/repos/kemalcr/kemal/releases/latest":
			http.NotFound(w, r)
		case "/repos/missing/missing":
			http.NotFound(w, r)
		case "/repos/empty-lang/demo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":             "demo",
				"full_name":        "empty-lang/demo",
				"html_url":         "https://github.com/empty-lang/demo",
				"language":         nil,
				"stargazers_count": 12,
				"topics":           []string{"framework"},
				"owner":            map[string]any{"login": "empty-lang", "type": "User"},
			})
		case "/repos/empty-lang/demo/languages":
			_ = json.NewEncoder(w).Encode(map[string]int{"Go": 800, "HTML": 20})
		case "/users/empty-lang":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "empty-lang", "type": "User", "location": "Ankara, Turkey"})
		case "/repos/empty-lang/demo/readme", "/repos/empty-lang/demo/releases/latest", "/repos/empty-lang/demo/contents/package.json", "/repos/empty-lang/demo/contents/composer.json":
			http.NotFound(w, r)
		case "/repos/kemalcr/kemal/contents/package.json", "/repos/kemalcr/kemal/contents/composer.json":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := gh.NewClient(gh.Options{BaseURL: srv.URL, Timeout: time.Second, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	seeds := []Seed{
		{Name: "Kemal", GitHub: "https://github.com/kemalcr/kemal", Category: "web-framework", InitialStatus: StatusVerified},
		{GitHub: "https://github.com/missing/missing", Category: "web-framework", InitialStatus: StatusVerified},
		{Name: "Demo", GitHub: "https://github.com/empty-lang/demo", Category: "web-framework", InitialStatus: StatusVerified},
	}
	cfg := &config.Config{
		Locations: []string{"Istanbul", "Ankara", "Turkey"},
		Countries: config.Countries{Countries: []config.Country{{Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish"}, Domains: []string{".com.tr"}}}},
	}
	items, rep, err := Scan(context.Background(), client, cfg, seeds, time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.RepositoryNotFound != 1 {
		t.Fatalf("not found=%d", rep.RepositoryNotFound)
	}
	var kemal, demo, missing Framework
	for _, item := range items {
		switch CanonicalKey(item.GitHub) {
		case "kemalcr/kemal":
			kemal = item
		case "empty-lang/demo":
			demo = item
		case "missing/missing":
			missing = item
		}
	}
	if kemal.Stars != 3901 {
		t.Fatalf("stars must come from API, got %d", kemal.Stars)
	}
	if kemal.Language != "Crystal" {
		t.Fatalf("language=%s", kemal.Language)
	}
	if kemal.License != "MIT" {
		t.Fatalf("license=%s", kemal.License)
	}
	if kemal.Website != "https://kemalcr.com" {
		t.Fatalf("website=%s", kemal.Website)
	}
	if kemal.Status != StatusVerified {
		t.Fatalf("status=%s score=%d evidence=%v", kemal.Status, kemal.CountryScore, kemal.CountryEvidence)
	}
	if len(kemal.CountryEvidence) == 0 {
		t.Fatal("expected country evidence")
	}
	if demo.Language != "Go" {
		t.Fatalf("language fallback got %s", demo.Language)
	}
	if missing.Status != StatusNotFound {
		t.Fatalf("missing status=%s", missing.Status)
	}
}

func TestCountryEvidenceUsesOwnerLocation(t *testing.T) {
	matcher := validation.NewCountryMatcher(&config.Config{
		Locations: []string{"Istanbul", "Turkey"},
		Countries: config.Countries{Countries: []config.Country{{Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish"}, Domains: []string{".com.tr"}}}},
	})
	evidence, score := countryEvidence(profileSignals{
		OwnerType: "Organization",
		Location:  "Istanbul, Turkey",
		IsOrg:     true,
	}, matcher)
	if score < 40 {
		t.Fatalf("score=%d", score)
	}
	if len(evidence) == 0 || evidence[0].Type != EvidenceOrganizationLocation {
		t.Fatalf("%+v", evidence)
	}
}

func TestResolveStatus(t *testing.T) {
	if got := resolveStatus(Seed{InitialStatus: StatusHistorical}, true, true, 90); got != StatusHistorical {
		t.Fatalf("%s", got)
	}
	if got := resolveStatus(Seed{InitialStatus: StatusVerified}, false, true, 90); got != StatusNotFound {
		t.Fatalf("%s", got)
	}
	if got := resolveStatus(Seed{InitialStatus: StatusPendingVerification}, true, true, 10); got != StatusPendingVerification {
		t.Fatalf("%s", got)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "config", "frameworks.yml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
