package project

import (
	"sort"
	"strings"
	"time"
)

const DatasetVersion = 1

type Dataset struct {
	Version     int       `json:"version"`
	GeneratedAt string    `json:"generated_at"`
	Projects    []Project `json:"projects"`
}

type Project struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Repository    string   `json:"repository,omitempty"`
	URL           string   `json:"url"`
	HTMLURL       string   `json:"html_url"`
	Owner         string   `json:"owner"`
	OwnerType     string   `json:"owner_type"`
	OwnerLocation string   `json:"owner_location,omitempty"`
	Description   string   `json:"description"`
	Language      string   `json:"language"`
	License       string   `json:"license"`
	LicenseStatus string   `json:"license_status"`
	Stars         int      `json:"stars"`
	Forks         int      `json:"forks"`
	OpenIssues    int      `json:"open_issues"`
	Topics        []string `json:"topics"`
	Category      string   `json:"category"`
	Categories    []string `json:"categories,omitempty"`
	Homepage      string   `json:"homepage,omitempty"`

	Country       string   `json:"country"`
	TurkeyScore   int      `json:"turkey_score"`
	ActivityScore int      `json:"activity_score"`
	QualityScore  int      `json:"quality_score"`
	TurkeySignals []string `json:"turkey_signals,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Source        string   `json:"source,omitempty"`
	Status        string   `json:"status,omitempty"`
	Verification  string   `json:"verification"`

	IsActive   bool `json:"is_active"`
	IsArchived bool `json:"is_archived"`
	IsFork     bool `json:"is_fork"`
	IsVerified bool `json:"is_verified"`

	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	PushedAt          string `json:"pushed_at,omitempty"`
	LastScannedAt     string `json:"last_scanned_at"`
	FirstDiscoveredAt string `json:"first_discovered_at"`

	StarsPrevious  int  `json:"stars_previous,omitempty"`
	StarDelta      int  `json:"star_delta"`
	ManualCategory bool `json:"manual_category,omitempty"`
}

func EmptyDataset(now time.Time) Dataset {
	return Dataset{
		Version:     DatasetVersion,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Projects:    []Project{},
	}
}

func SortProjects(projects []Project) {
	sort.SliceStable(projects, func(i, j int) bool {
		a, b := projects[i], projects[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.Stars != b.Stars {
			return a.Stars > b.Stars
		}
		if !strings.EqualFold(a.Name, b.Name) {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return strings.ToLower(a.FullName) < strings.ToLower(b.FullName)
	})
}

func IndexByID(projects []Project) map[int64]Project {
	out := make(map[int64]Project, len(projects))
	for _, p := range projects {
		out[p.ID] = p
	}
	return out
}

func CloneTopics(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	return uniqueSorted(out)
}

func CloneStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := append([]string(nil), in...)
	return out
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func IsCommunityVerified(p Project) bool {
	return p.IsVerified && strings.EqualFold(p.Verification, "community")
}
