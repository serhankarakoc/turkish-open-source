package framework

import (
	"sort"
	"strings"
	"time"
)

const DatasetVersion = 1

const (
	StatusVerified            = "verified"
	StatusPendingVerification = "pending_verification"
	StatusHistorical          = "historical"
	StatusNotFound            = "repository_not_found"
	StatusExcluded            = "excluded"
)

type Dataset struct {
	Version     int         `json:"version"`
	GeneratedAt string      `json:"generated_at"`
	Frameworks  []Framework `json:"frameworks"`
}

type Framework struct {
	Name            string     `json:"name"`
	Language        string     `json:"language"`
	Category        string     `json:"category"`
	GitHub          string     `json:"github"`
	Website         string     `json:"website"`
	Stars           int        `json:"stars"`
	License         string     `json:"license"`
	CountryEvidence []Evidence `json:"country_evidence"`
	Status          string     `json:"status"`

	GitHubOwner   string   `json:"github_owner,omitempty"`
	GitHubRepo    string   `json:"github_repo,omitempty"`
	CountryScore  int      `json:"country_score,omitempty"`
	Forks         int      `json:"forks,omitempty"`
	OpenIssues    int      `json:"open_issues,omitempty"`
	Watchers      int      `json:"watchers,omitempty"`
	DefaultBranch string   `json:"default_branch,omitempty"`
	Description   string   `json:"description,omitempty"`
	Topics        []string `json:"topics,omitempty"`
	LastCommit    string   `json:"last_commit,omitempty"`
	LastRelease   string   `json:"last_release,omitempty"`
	Archived      bool     `json:"archived,omitempty"`
	Fork          bool     `json:"fork,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
	ScannedAt     string   `json:"scanned_at,omitempty"`
}

type Evidence struct {
	Type   string `json:"type"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

func EmptyDataset(now time.Time) Dataset {
	stamp := ""
	if !now.IsZero() {
		stamp = now.UTC().Format(time.RFC3339)
	}
	return Dataset{
		Version:     DatasetVersion,
		GeneratedAt: stamp,
		Frameworks:  []Framework{},
	}
}

func SortFrameworks(items []Framework) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.Stars != b.Stars {
			return a.Stars > b.Stars
		}
		if !strings.EqualFold(a.Name, b.Name) {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return strings.ToLower(a.GitHub) < strings.ToLower(b.GitHub)
	})
}

func CanonicalKey(githubURL string) string {
	owner, repo, _, err := ParseGitHubRepo(githubURL)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(githubURL))
	}
	return strings.ToLower(owner + "/" + repo)
}
