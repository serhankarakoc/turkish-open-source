package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Root       string
	Scanner    ScannerConfig
	Search     SearchConfig
	Projects   ProjectsConfig
	Enrichment EnrichmentConfig
	GitHub     GitHubConfig
	Readme     ReadmeConfig
	Categories map[string]Category
	Keywords   Keywords
	Locations  []string
	Countries  Countries
}

type ScannerConfig struct {
	MaxWorkers               int `yaml:"max_workers"`
	MaxRetries               int `yaml:"max_retries"`
	RequestTimeoutSeconds    int `yaml:"request_timeout_seconds"`
	RetryBackoffMilliseconds int `yaml:"retry_backoff_milliseconds"`
	MaxBackoffSeconds        int `yaml:"max_backoff_seconds"`
}

type SearchConfig struct {
	MaxPagesPerQuery int `yaml:"max_pages_per_query"`
	ResultsPerPage   int `yaml:"results_per_page"`
	MaxReposPerUser  int `yaml:"max_repos_per_user"`
	MaxUsersPerScan  int `yaml:"max_users_per_scan"`
	MaxOrgsPerScan   int `yaml:"max_orgs_per_scan"`
}

type ProjectsConfig struct {
	MinimumTurkeyScore           int  `yaml:"minimum_turkey_score"`
	MinimumActivityScore         int  `yaml:"minimum_activity_score"`
	MinimumQualityScore          int  `yaml:"minimum_quality_score"`
	MinimumStars                 int  `yaml:"minimum_stars"`
	AllowLowStarException        bool `yaml:"allow_low_star_exception"`
	LowStarExceptionTurkeyScore  int  `yaml:"low_star_exception_turkey_score"`
	LowStarExceptionQualityScore int  `yaml:"low_star_exception_quality_score"`
	IncludeArchived              bool `yaml:"include_archived"`
	IncludeForks                 bool `yaml:"include_forks"`
	RequireLicense               bool `yaml:"require_license"`
	ShowUnlicensed               bool `yaml:"show_unlicensed"`
}

type EnrichmentConfig struct {
	FetchOwner        bool `yaml:"fetch_owner"`
	FetchReadme       bool `yaml:"fetch_readme"`
	FetchRelease      bool `yaml:"fetch_release"`
	FetchContributors bool `yaml:"fetch_contributors"`
}

type GitHubConfig struct {
	APIVersion string `yaml:"api_version"`
	APIURL     string `yaml:"api_url"`
}

type ReadmeConfig struct {
	TrendingLimit int `yaml:"trending_limit"`
	StarredLimit  int `yaml:"starred_limit"`
	RecentLimit   int `yaml:"recent_limit"`
	CategoryLimit int `yaml:"category_limit"`
}

type Category struct {
	Key    string
	Label  string   `yaml:"label"`
	Emoji  string   `yaml:"emoji"`
	Topics []string `yaml:"topics"`
}

type Keywords struct {
	Keywords         []string            `yaml:"keywords"`
	Topics           []string            `yaml:"topics"`
	CountryLocations []string            `yaml:"country_locations"`
	Languages        []string            `yaml:"languages"`
	IntentKeywords   map[string][]string `yaml:"intent_keywords"`
	SearchPhrases    []string            `yaml:"search_phrases"`
	PriorityTopics   []string            `yaml:"priority_topics"`
	TurkishStopwords []string            `yaml:"turkish_stopwords"`
}

type Countries struct {
	Version   int       `json:"version"`
	Default   string    `json:"default"`
	Countries []Country `json:"countries"`
}

type Country struct {
	Code       string   `json:"code"`
	Names      []string `json:"names"`
	Adjectives []string `json:"adjectives"`
	Domains    []string `json:"domains"`
}

type settingsFile struct {
	Scanner    ScannerConfig    `yaml:"scanner"`
	Search     SearchConfig     `yaml:"search"`
	Projects   ProjectsConfig   `yaml:"projects"`
	Enrichment EnrichmentConfig `yaml:"enrichment"`
	GitHub     GitHubConfig     `yaml:"github"`
	Readme     ReadmeConfig     `yaml:"readme"`
}

type categoriesFile struct {
	Categories map[string]Category `yaml:"categories"`
}

type citiesFile struct {
	Locations []string `yaml:"locations"`
}

func Load(root string) (*Config, error) {
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = wd
	}
	cfg := defaultConfig()
	cfg.Root = root

	var settings settingsFile
	if err := readYAML(filepath.Join(root, "config", "settings.yml"), &settings); err != nil {
		return nil, err
	}
	cfg.Scanner = settings.Scanner
	cfg.Search = settings.Search
	cfg.Projects = settings.Projects
	cfg.Enrichment = settings.Enrichment
	cfg.GitHub = settings.GitHub
	cfg.Readme = settings.Readme
	applyDefaults(cfg)

	var cats categoriesFile
	if err := readYAML(filepath.Join(root, "config", "categories.yml"), &cats); err != nil {
		return nil, err
	}
	cfg.Categories = map[string]Category{}
	for key, cat := range cats.Categories {
		cat.Key = key
		if cat.Label == "" {
			cat.Label = key
		}
		cfg.Categories[key] = cat
	}

	if err := readYAML(filepath.Join(root, "config", "keywords.yml"), &cfg.Keywords); err != nil {
		return nil, err
	}

	var cities citiesFile
	if err := readYAML(filepath.Join(root, "config", "cities.yml"), &cities); err != nil {
		return nil, err
	}
	cfg.Locations = cities.Locations

	raw, err := os.ReadFile(filepath.Join(root, "data", "countries.json"))
	if err != nil {
		return nil, fmt.Errorf("read countries.json: %w", err)
	}
	if err := json.Unmarshal(raw, &cfg.Countries); err != nil {
		return nil, fmt.Errorf("parse countries.json: %w", err)
	}

	if v := os.Getenv("GITHUB_API_URL"); v != "" {
		cfg.GitHub.APIURL = v
	}
	if v := os.Getenv("GITHUB_API_VERSION"); v != "" {
		cfg.GitHub.APIVersion = v
	}
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Scanner: ScannerConfig{
			MaxWorkers:               6,
			MaxRetries:               5,
			RequestTimeoutSeconds:    20,
			RetryBackoffMilliseconds: 500,
			MaxBackoffSeconds:        60,
		},
		Search: SearchConfig{
			MaxPagesPerQuery: 5,
			ResultsPerPage:   100,
			MaxReposPerUser:  100,
			MaxUsersPerScan:  250,
			MaxOrgsPerScan:   150,
		},
		Projects: ProjectsConfig{
			MinimumTurkeyScore:           75,
			MinimumActivityScore:         0,
			MinimumQualityScore:          25,
			MinimumStars:                 10,
			AllowLowStarException:        true,
			LowStarExceptionTurkeyScore:  85,
			LowStarExceptionQualityScore: 55,
			RequireLicense:               true,
		},
		Enrichment: EnrichmentConfig{
			FetchOwner:   true,
			FetchReadme:  true,
			FetchRelease: true,
		},
		GitHub: GitHubConfig{
			APIVersion: "2026-03-10",
			APIURL:     "https://api.github.com",
		},
		Readme: ReadmeConfig{
			TrendingLimit: 10,
			StarredLimit:  15,
			RecentLimit:   10,
			CategoryLimit: 40,
		},
		Categories: map[string]Category{},
	}
}

func applyDefaults(cfg *Config) {
	d := defaultConfig()
	if cfg.Scanner.MaxWorkers <= 0 {
		cfg.Scanner.MaxWorkers = d.Scanner.MaxWorkers
	}
	if cfg.Scanner.MaxWorkers > 8 {
		cfg.Scanner.MaxWorkers = 8
	}
	if cfg.Scanner.MaxRetries < 0 {
		cfg.Scanner.MaxRetries = d.Scanner.MaxRetries
	}
	if cfg.Scanner.RequestTimeoutSeconds <= 0 {
		cfg.Scanner.RequestTimeoutSeconds = d.Scanner.RequestTimeoutSeconds
	}
	if cfg.Scanner.RetryBackoffMilliseconds <= 0 {
		cfg.Scanner.RetryBackoffMilliseconds = d.Scanner.RetryBackoffMilliseconds
	}
	if cfg.Scanner.MaxBackoffSeconds <= 0 {
		cfg.Scanner.MaxBackoffSeconds = d.Scanner.MaxBackoffSeconds
	}
	if cfg.Search.MaxPagesPerQuery <= 0 {
		cfg.Search.MaxPagesPerQuery = d.Search.MaxPagesPerQuery
	}
	if cfg.Search.ResultsPerPage <= 0 || cfg.Search.ResultsPerPage > 100 {
		cfg.Search.ResultsPerPage = 100
	}
	if cfg.Search.MaxReposPerUser <= 0 {
		cfg.Search.MaxReposPerUser = d.Search.MaxReposPerUser
	}
	if cfg.Search.MaxUsersPerScan <= 0 {
		cfg.Search.MaxUsersPerScan = d.Search.MaxUsersPerScan
	}
	if cfg.Search.MaxOrgsPerScan <= 0 {
		cfg.Search.MaxOrgsPerScan = d.Search.MaxOrgsPerScan
	}
	if cfg.Projects.MinimumStars < 0 {
		cfg.Projects.MinimumStars = d.Projects.MinimumStars
	}
	if cfg.Projects.MinimumTurkeyScore <= 0 {
		cfg.Projects.MinimumTurkeyScore = d.Projects.MinimumTurkeyScore
	}
	if cfg.Projects.MinimumActivityScore < 0 {
		cfg.Projects.MinimumActivityScore = d.Projects.MinimumActivityScore
	}
	if cfg.Projects.MinimumQualityScore < 0 {
		cfg.Projects.MinimumQualityScore = d.Projects.MinimumQualityScore
	}
	if cfg.Projects.LowStarExceptionTurkeyScore <= 0 {
		cfg.Projects.LowStarExceptionTurkeyScore = d.Projects.LowStarExceptionTurkeyScore
	}
	if cfg.Projects.LowStarExceptionQualityScore <= 0 {
		cfg.Projects.LowStarExceptionQualityScore = d.Projects.LowStarExceptionQualityScore
	}
	if cfg.GitHub.APIURL == "" {
		cfg.GitHub.APIURL = d.GitHub.APIURL
	}
	if cfg.GitHub.APIVersion == "" {
		cfg.GitHub.APIVersion = d.GitHub.APIVersion
	}
	if cfg.Readme.TrendingLimit <= 0 {
		cfg.Readme.TrendingLimit = d.Readme.TrendingLimit
	}
	if cfg.Readme.StarredLimit <= 0 {
		cfg.Readme.StarredLimit = d.Readme.StarredLimit
	}
	if cfg.Readme.RecentLimit <= 0 {
		cfg.Readme.RecentLimit = d.Readme.RecentLimit
	}
	if cfg.Readme.CategoryLimit <= 0 {
		cfg.Readme.CategoryLimit = d.Readme.CategoryLimit
	}
}

func readYAML(path string, dest any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func (c *Config) CategoryKeys() map[string]struct{} {
	out := make(map[string]struct{}, len(c.Categories))
	for k := range c.Categories {
		out[k] = struct{}{}
	}
	out["other"] = struct{}{}
	return out
}
