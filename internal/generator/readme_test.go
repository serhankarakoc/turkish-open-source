package generator

import (
	"strings"
	"testing"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/framework"
	"github.com/serhankarakoc/turkish-open-source/internal/project"
)

func sampleProjects() []project.Project {
	return []project.Project{
		{
			ID: 2, Name: "Beta", FullName: "acme/beta", HTMLURL: "https://github.com/acme/beta",
			Language: "Python", Stars: 10, Forks: 1, License: "Apache-2.0", Category: "web",
			IsActive: true, FirstDiscoveredAt: "2026-08-01T00:00:00Z", StarDelta: 3,
		},
		{
			ID: 1, Name: "Alpha", FullName: "acme/alpha", HTMLURL: "https://github.com/acme/alpha",
			Language: "Go", Stars: 50, Forks: 4, License: "MIT", Category: "devtools",
			IsActive: true, FirstDiscoveredAt: "2026-08-10T00:00:00Z", StarDelta: 1,
		},
		{
			ID: 3, Name: "Old", FullName: "acme/old", HTMLURL: "https://github.com/acme/old",
			Language: "Go", Stars: 80, License: "MIT", Category: "devtools",
			IsActive: false, IsArchived: true,
		},
	}
}

func TestGenerateContainsMarkersAndLinks(t *testing.T) {
	cfg := &config.Config{
		Categories: map[string]config.Category{
			"devtools": {Key: "devtools", Label: "Developer Tools", Emoji: "🛠️"},
			"web":      {Key: "web", Label: "Web", Emoji: "🌐"},
		},
		Readme: config.ReadmeConfig{TrendingLimit: 10, StarredLimit: 10, RecentLimit: 10, CategoryLimit: 10},
	}
	now := time.Date(2026, 8, 17, 2, 0, 0, 0, time.UTC)
	out := Generate(sampleProjects(), nil, cfg, now)
	for _, needle := range []string{GeneratedStart, GeneratedEnd, "## Framework'ler"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q", needle)
		}
	}
	if strings.Contains(out, "ZATRANO") {
		t.Fatal("must not invent catalog entries")
	}
}

func TestPatchPreservesManualPreamble(t *testing.T) {
	existing := "# Custom\n\n" + GeneratedStart + "\nPLACEHOLDER_BODY\n" + GeneratedEnd + "\n"
	cfg := &config.Config{Readme: config.ReadmeConfig{TrendingLimit: 5, StarredLimit: 5, RecentLimit: 5, CategoryLimit: 5}}
	out := Patch(existing, sampleProjects(), nil, cfg, time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC))
	if !strings.HasPrefix(out, "# Custom\n") {
		t.Fatal("preamble lost")
	}
	if strings.Contains(out, "PLACEHOLDER_BODY") {
		t.Fatal("old generated body should be replaced")
	}
}

func TestArchivedExcludedFromActiveRankings(t *testing.T) {
	cfg := &config.Config{
		Categories: map[string]config.Category{"devtools": {Label: "Developer Tools"}},
		Readme:     config.ReadmeConfig{StarredLimit: 10, TrendingLimit: 10, RecentLimit: 10, CategoryLimit: 10},
	}
	out := Generate(sampleProjects(), nil, cfg, time.Now().UTC())
	if strings.Contains(out, "Old") {
		t.Fatal("archived project leaked into README")
	}
	if strings.Contains(out, "## En çok yıldız alan projeler") {
		t.Fatal("starred catalog section should be removed")
	}
}

func TestGenerateDeterministic(t *testing.T) {
	cfg := &config.Config{Readme: config.ReadmeConfig{StarredLimit: 10, TrendingLimit: 10, RecentLimit: 10, CategoryLimit: 10}}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	a := Generate(sampleProjects(), nil, cfg, now)
	b := Generate(sampleProjects(), nil, cfg, now)
	if a != b {
		t.Fatal("README generation must be deterministic")
	}
}

func TestDetectCategory(t *testing.T) {
	cats := map[string]config.Category{
		"ai":    {Topics: []string{"llm", "ai"}},
		"web":   {Topics: []string{"frontend", "react"}},
		"other": {},
	}
	if got := DetectCategory([]string{"llm", "rag"}, "Python", cats); got != "ai" {
		t.Fatalf("got %s", got)
	}
	if got := DetectCategory(nil, "Swift", map[string]config.Category{"mobile": {}}); got != "mobile" {
		t.Fatalf("language fallback got %s", got)
	}
	if got := DetectCategory(nil, "Go", cats); got != "other" {
		t.Fatalf("fallback got %s", got)
	}
}

func TestComputeStatistics(t *testing.T) {
	st := ComputeStatistics(sampleProjects())
	if st.TotalProjects != 3 || st.ActiveProjects != 2 || st.ArchivedProjects != 1 {
		t.Fatalf("%+v", st)
	}
	if st.Languages != 2 {
		t.Fatalf("languages=%d", st.Languages)
	}
}

func TestGenerateIncludesFrameworkTable(t *testing.T) {
	cfg := &config.Config{Readme: config.ReadmeConfig{StarredLimit: 10, TrendingLimit: 10, RecentLimit: 10, CategoryLimit: 10}}
	frameworks := []framework.Framework{
		{Name: "Kemal", Language: "Crystal", Category: "web-framework", GitHub: "https://github.com/kemalcr/kemal", Stars: 3901, License: "MIT", Status: "verified"},
		{Name: "Missing", GitHub: "https://github.com/missing/missing", Status: framework.StatusNotFound},
	}
	out := Generate(sampleProjects(), frameworks, cfg, time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC))
	for _, needle := range []string{"## Framework'ler", "[Kemal](https://github.com/kemalcr/kemal)", "Crystal", "3901", "Web"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q", needle)
		}
	}
	if strings.Contains(out, "Missing") || strings.Contains(out, "repository_not_found") {
		t.Fatal("not-found frameworks must not appear in README")
	}
	if strings.Contains(out, "## Statistics") || strings.Contains(out, "## Trending") || strings.Contains(out, "Top Frameworks, Libraries") {
		t.Fatal("removed noisy sections leaked into README")
	}
}
