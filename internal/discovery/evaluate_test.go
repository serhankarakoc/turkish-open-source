package discovery

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
)

func TestEvaluateRejectsForksAndUnlicensed(t *testing.T) {
	cfg := &config.Config{
		Scanner:    config.ScannerConfig{MaxWorkers: 2},
		Projects:   config.ProjectsConfig{MinimumTurkeyScore: 75, MinimumStars: 10, RequireLicense: true},
		Categories: map[string]config.Category{"other": {Key: "other", Label: "Other"}},
	}
	set := NewSet()
	set.AddRepository(gh.Repository{
		ID: 1, Name: "forked", FullName: "a/forked", Fork: true,
		HTMLURL: "https://github.com/a/forked",
		License: &gh.License{SPDXID: "MIT"},
	}, "t")
	set.AddRepository(gh.Repository{
		ID: 2, Name: "nolicense", FullName: "a/nolicense",
		HTMLURL: "https://github.com/a/nolicense",
	}, "t")
	got, stats, err := Evaluate(context.Background(), nil, cfg, set.Candidates(), time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("accepted %+v", got)
	}
	if stats.Rejected != 2 {
		t.Fatalf("rejected=%d", stats.Rejected)
	}
}

func TestEvaluateAcceptsStrongTurkishProject(t *testing.T) {
	cfg := &config.Config{
		Scanner: config.ScannerConfig{MaxWorkers: 2},
		Projects: config.ProjectsConfig{
			MinimumTurkeyScore:  50,
			MinimumQualityScore: 0,
			MinimumStars:        10,
			RequireLicense:      true,
		},
		Locations: []string{"Istanbul", "Turkey"},
		Keywords:  config.Keywords{TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane", "uygulama"}},
		Countries: config.Countries{Countries: []config.Country{{
			Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish"}, Domains: []string{".tr"},
		}}},
		Categories: map[string]config.Category{
			"devtools": {Key: "devtools", Topics: []string{"cli"}},
			"other":    {Key: "other"},
		},
		Enrichment: config.EnrichmentConfig{},
	}
	set := NewSet()
	set.AddRepository(gh.Repository{
		ID: 9, Name: "cli", FullName: "tr/cli", HTMLURL: "https://github.com/tr/cli", URL: "https://api.github.com/repos/tr/cli",
		Description:     "Bu kütüphane ve yazılım uygulaması için geliştirildi.",
		Language:        "Go",
		Topics:          []string{"cli", "made-in-turkey"},
		Homepage:        "https://cli.example.com.tr",
		StargazersCount: 12,
		ForksCount:      3,
		License:         &gh.License{SPDXID: "MIT"},
		Owner:           &gh.User{Login: "tr", Type: "User", Location: "Istanbul, Turkey"},
		PushedAt:        time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC().Add(-40 * 24 * time.Hour),
	}, "topic:turkey")
	got, stats, err := Evaluate(context.Background(), nil, cfg, set.Candidates(), time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("accepted=%d stats=%+v", len(got), stats)
	}
	if got[0].LicenseStatus != "valid" || got[0].Category == "" {
		t.Fatalf("%+v", got[0])
	}
	if got[0].Status == "" || got[0].QualityScore == 0 || len(got[0].Evidence) == 0 {
		t.Fatalf("missing new scoring fields: %+v", got[0])
	}
}

func TestEvaluateRejectsLowStars(t *testing.T) {
	cfg := &config.Config{
		Scanner: config.ScannerConfig{MaxWorkers: 2},
		Projects: config.ProjectsConfig{
			MinimumTurkeyScore:  50,
			MinimumQualityScore: 0,
			MinimumStars:        10,
			RequireLicense:      true,
		},
		Locations: []string{"Istanbul", "Turkey"},
		Keywords:  config.Keywords{TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane", "uygulama"}},
		Countries: config.Countries{Countries: []config.Country{{
			Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish"}, Domains: []string{".tr"},
		}}},
		Categories: map[string]config.Category{
			"devtools": {Key: "devtools", Topics: []string{"cli"}},
			"other":    {Key: "other"},
		},
		Enrichment: config.EnrichmentConfig{},
	}

	set := NewSet()
	set.AddRepository(gh.Repository{
		ID:              11,
		Name:            "cli-low-stars",
		FullName:        "tr/cli-low-stars",
		HTMLURL:         "https://github.com/tr/cli-low-stars",
		URL:             "https://api.github.com/repos/tr/cli-low-stars",
		Description:     "Bu kütüphane ve yazılım uygulaması için geliştirildi.",
		Language:        "Go",
		Topics:          []string{"cli", "made-in-turkey"},
		Homepage:        "https://cli-low.example.com.tr",
		StargazersCount: 1,
		License:         &gh.License{SPDXID: "MIT"},
		Owner:           &gh.User{Login: "tr", Type: "User", Location: "Istanbul, Turkey"},
		PushedAt:        time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC().Add(-40 * 24 * time.Hour),
	}, "topic:turkey")

	got, stats, err := Evaluate(context.Background(), nil, cfg, set.Candidates(), time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("accepted %+v", got)
	}
	if stats.Rejected != 1 {
		t.Fatalf("rejected=%d", stats.Rejected)
	}
}

func TestEvaluateAllowsLowStarExceptionForFramework(t *testing.T) {
	cfg := &config.Config{
		Scanner: config.ScannerConfig{MaxWorkers: 2},
		Projects: config.ProjectsConfig{
			MinimumTurkeyScore:           75,
			MinimumQualityScore:          0,
			MinimumStars:                 10,
			AllowLowStarException:        true,
			LowStarExceptionTurkeyScore:  85,
			LowStarExceptionQualityScore: 20,
			RequireLicense:               true,
		},
		Locations: []string{"Istanbul", "Turkey"},
		Keywords: config.Keywords{
			TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane", "uygulama"},
			PriorityTopics:   []string{"framework", "fiber", "auth"},
		},
		Countries: config.Countries{Countries: []config.Country{{
			Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish"}, Domains: []string{".tr"},
		}}},
		Categories: map[string]config.Category{
			"framework": {Key: "framework", Topics: []string{"framework", "fiber"}},
			"other":     {Key: "other"},
		},
		Enrichment: config.EnrichmentConfig{},
	}

	set := NewSet()
	set.AddRepository(gh.Repository{
		ID:              21,
		Name:            "z-framework",
		FullName:        "tr/z-framework",
		HTMLURL:         "https://github.com/tr/z-framework",
		URL:             "https://api.github.com/repos/tr/z-framework",
		Description:     "Bu framework Türkiye'de geliştirilen bir yazılım projesidir.",
		Language:        "Go",
		Topics:          []string{"framework", "fiber", "auth", "made-in-turkey"},
		Homepage:        "https://framework.example.com.tr",
		StargazersCount: 3,
		ForksCount:      2,
		License:         &gh.License{SPDXID: "MIT"},
		Owner:           &gh.User{Login: "tr", Type: "Organization", Location: "Istanbul, Turkey", Company: "Turkey"},
		PushedAt:        time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC().Add(-40 * 24 * time.Hour),
	}, "intent:framework")

	got, _, err := Evaluate(context.Background(), nil, cfg, set.Candidates(), time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected low-star exception acceptance, got %+v", got)
	}
}

func TestEvaluateFindsZATRANOStyleFramework(t *testing.T) {
	assertFrameworkFixtureAccepted(t, "zatrano/framework", "Organization", "Türkiye", []string{"framework", "fiber", "auth"}, "https://www.zatrano.com.tr", 3)
}

func TestEvaluateFindsKemalStyleFramework(t *testing.T) {
	assertFrameworkFixtureAccepted(t, "kemalcr/kemal", "Organization", "Istanbul, Turkey", []string{"framework", "crystal", "web-framework"}, "https://kemalcr.com", 1200)
}

func TestEvaluateFindsZNFrameworkStyleProject(t *testing.T) {
	assertFrameworkFixtureAccepted(t, "znframework/znframework", "Organization", "Türkiye", []string{"framework", "php", "web-framework"}, "https://znframework.com.tr", 40)
}

func TestEvaluateFindsPrimeStyleProject(t *testing.T) {
	assertFrameworkFixtureAccepted(t, "prime-software/prime", "Organization", "Ankara, Turkey", []string{"framework", "go", "api"}, "https://prime.example.com.tr", 12)
}

func assertFrameworkFixtureAccepted(t *testing.T, fullName, ownerType, ownerLocation string, topics []string, homepage string, stars int) {
	t.Helper()
	cfg := &config.Config{
		Scanner: config.ScannerConfig{MaxWorkers: 2},
		Projects: config.ProjectsConfig{
			MinimumTurkeyScore:           75,
			MinimumQualityScore:          0,
			MinimumStars:                 10,
			AllowLowStarException:        true,
			LowStarExceptionTurkeyScore:  85,
			LowStarExceptionQualityScore: 20,
			RequireLicense:               true,
		},
		Locations: []string{"Istanbul", "Ankara", "Turkey", "Türkiye"},
		Keywords: config.Keywords{
			TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane", "uygulama", "geliştirildi"},
			PriorityTopics:   []string{"framework", "fiber", "auth", "api", "web-framework"},
		},
		Countries: config.Countries{Countries: []config.Country{{
			Names: []string{"Turkey", "Türkiye"}, Adjectives: []string{"Turkish", "Türk"}, Domains: []string{".tr"},
		}}},
		Categories: map[string]config.Category{
			"framework": {Key: "framework", Topics: []string{"framework", "fiber", "api", "web-framework"}},
			"other":     {Key: "other"},
		},
		Enrichment: config.EnrichmentConfig{},
	}
	parts := strings.SplitN(fullName, "/", 2)
	now := time.Now().UTC()
	set := NewSet()
	set.AddRepository(gh.Repository{
		ID:              int64(stars + len(fullName)),
		Name:            parts[1],
		FullName:        fullName,
		HTMLURL:         "https://github.com/" + fullName,
		URL:             "https://api.github.com/repos/" + fullName,
		Description:     "Bu framework Türkiye'de geliştirilen açık kaynak bir yazılım projesidir.",
		Language:        "Go",
		Topics:          topics,
		Homepage:        homepage,
		StargazersCount: stars,
		ForksCount:      2,
		License:         &gh.License{SPDXID: "MIT"},
		Owner:           &gh.User{Login: parts[0], Type: ownerType, Location: ownerLocation, Company: "Turkey", Blog: homepage},
		PushedAt:        now,
		UpdatedAt:       now,
		CreatedAt:       now.Add(-40 * 24 * time.Hour),
	}, "fixture")
	got, _, err := Evaluate(context.Background(), nil, cfg, set.Candidates(), now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected %s to be accepted, got %+v", fullName, got)
	}
	if got[0].Category != "framework" || got[0].Status == "" || got[0].TurkeyScore < 75 {
		t.Fatalf("unexpected scored project: %+v", got[0])
	}
}
