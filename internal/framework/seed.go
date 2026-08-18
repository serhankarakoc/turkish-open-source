package framework

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Seed struct {
	Name          string `yaml:"name"`
	GitHub        string `yaml:"github"`
	Category      string `yaml:"category"`
	InitialStatus string `yaml:"initial_status"`
}

type seedFile struct {
	Frameworks []Seed `yaml:"frameworks"`
}

func LoadSeeds(root string) ([]Seed, error) {
	path := filepath.Join(root, "config", "frameworks.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var file seedFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make([]Seed, 0, len(file.Frameworks))
	seen := map[string]struct{}{}
	for i, seed := range file.Frameworks {
		seed.GitHub = strings.TrimSpace(seed.GitHub)
		seed.Category = strings.TrimSpace(seed.Category)
		seed.InitialStatus = strings.TrimSpace(seed.InitialStatus)
		seed.Name = strings.TrimSpace(seed.Name)
		if seed.GitHub == "" {
			return nil, fmt.Errorf("%s: framework %d missing github", path, i+1)
		}
		if seed.Category == "" {
			return nil, fmt.Errorf("%s: %s missing category", path, seed.GitHub)
		}
		if seed.InitialStatus == "" {
			seed.InitialStatus = StatusPendingVerification
		}
		if !validSeedStatus(seed.InitialStatus) {
			return nil, fmt.Errorf("%s: %s has invalid initial_status %q", path, seed.GitHub, seed.InitialStatus)
		}
		key := CanonicalKey(seed.GitHub)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%s: duplicate github %s", path, seed.GitHub)
		}
		seen[key] = struct{}{}
		out = append(out, seed)
	}
	return out, nil
}

func ParseGitHubRepo(raw string) (owner, repo, canonical string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", fmt.Errorf("empty github url")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimPrefix(raw, "//")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("github url: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", "", "", fmt.Errorf("not a github.com url: %s", raw)
	}
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("github url must be https://github.com/owner/repo: %s", raw)
	}
	if parts[0] == "orgs" || parts[0] == "users" || parts[0] == "settings" || parts[0] == "topics" {
		return "", "", "", fmt.Errorf("github url must point at a repository: %s", raw)
	}
	owner = parts[0]
	repo = parts[1]
	canonical = "https://github.com/" + owner + "/" + repo
	return owner, repo, canonical, nil
}

func validSeedStatus(status string) bool {
	switch status {
	case StatusVerified, StatusPendingVerification, StatusHistorical, StatusExcluded:
		return true
	default:
		return false
	}
}
