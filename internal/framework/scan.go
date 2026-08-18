package framework

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

type Logger interface {
	Printf(format string, v ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

type Report struct {
	Total               int
	Verified            int
	PendingVerification int
	Historical          int
	RepositoryNotFound  int
	Excluded            int
	InvalidURLs         int
	Frameworks          []Framework
}

type ownerCache struct {
	mu sync.Mutex
	m  map[string]*gh.User
}

func newOwnerCache() *ownerCache {
	return &ownerCache{m: map[string]*gh.User{}}
}

func (c *ownerCache) get(login string) (*gh.User, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, ok := c.m[strings.ToLower(login)]
	return u, ok
}

func (c *ownerCache) put(u *gh.User) {
	if u == nil || u.Login == "" {
		return
	}
	c.mu.Lock()
	c.m[strings.ToLower(u.Login)] = u
	c.mu.Unlock()
}

func Scan(ctx context.Context, client *gh.Client, cfg *config.Config, seeds []Seed, now time.Time, log Logger) ([]Framework, Report, error) {
	if log == nil {
		log = nopLogger{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	matcher := validation.NewCountryMatcher(cfg)
	owners := newOwnerCache()
	out := make([]Framework, 0, len(seeds))
	rep := Report{}
	seen := map[string]struct{}{}

	for _, seed := range seeds {
		if err := ctx.Err(); err != nil {
			return out, rep, err
		}
		owner, repo, canonical, err := ParseGitHubRepo(seed.GitHub)
		if err != nil {
			log.Printf("framework seed %s: %v", seed.GitHub, err)
			rep.InvalidURLs++
			item := missingFramework(seed, "", err.Error(), now)
			item.Status = StatusNotFound
			out = append(out, item)
			countStatus(&rep, item.Status)
			continue
		}
		key := strings.ToLower(owner + "/" + repo)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		item, err := scanOne(ctx, client, matcher, owners, seed, owner, repo, canonical, now, log)
		if err != nil {
			if gh.IsRateLimited(err) {
				return out, rep, fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN for a higher quota: %w", err)
			}
			return out, rep, err
		}
		out = append(out, item)
		countStatus(&rep, item.Status)
	}

	SortFrameworks(out)
	rep.Total = len(out)
	rep.Frameworks = out
	return out, rep, nil
}

func scanOne(ctx context.Context, client *gh.Client, matcher *validation.CountryMatcher, owners *ownerCache, seed Seed, owner, repo, canonical string, now time.Time, log Logger) (Framework, error) {
	item := Framework{
		Name:            displayName(seed, repo),
		Category:        seed.Category,
		GitHub:          canonical,
		GitHubOwner:     owner,
		GitHubRepo:      repo,
		License:         "Unknown",
		CountryEvidence: []Evidence{},
		Status:          StatusNotFound,
		ScannedAt:       now.UTC().Format(time.RFC3339),
	}

	repoMeta, err := client.GetRepository(ctx, owner, repo)
	if err != nil {
		if gh.IsNotFound(err) {
			log.Printf("framework %s/%s: repository not found", owner, repo)
			item.Status = StatusNotFound
			return item, nil
		}
		if gh.IsRateLimited(err) {
			return item, err
		}
		log.Printf("framework %s/%s: %v", owner, repo, err)
		item.Status = StatusNotFound
		return item, nil
	}

	if repoMeta.HTMLURL != "" {
		item.GitHub = strings.TrimRight(repoMeta.HTMLURL, "/")
	}
	item.GitHubOwner = repoMeta.OwnerLogin()
	item.GitHubRepo = repoMeta.Name
	item.Name = displayName(seed, repoMeta.Name)
	item.Language = strings.TrimSpace(repoMeta.Language)
	item.Stars = repoMeta.StargazersCount
	item.Forks = repoMeta.ForksCount
	item.OpenIssues = repoMeta.OpenIssuesCount
	item.Watchers = repoMeta.SubscribersCount
	if item.Watchers == 0 {
		item.Watchers = repoMeta.WatchersCount
	}
	item.DefaultBranch = repoMeta.DefaultBranch
	item.Description = strings.TrimSpace(repoMeta.Description)
	item.Topics = append([]string(nil), repoMeta.Topics...)
	item.Archived = repoMeta.Archived
	item.Fork = repoMeta.Fork
	item.CreatedAt = formatTime(repoMeta.CreatedAt)
	item.UpdatedAt = formatTime(repoMeta.UpdatedAt)
	item.LastCommit = formatTime(repoMeta.PushedAt)
	if spdx := strings.TrimSpace(repoMeta.LicenseID()); spdx != "" {
		item.License = spdx
	} else {
		item.License = "Unknown"
	}

	if item.Language == "" {
		langs, langErr := client.GetLanguages(ctx, owner, repo)
		if langErr != nil && gh.IsRateLimited(langErr) {
			return item, langErr
		}
		if langErr != nil {
			log.Printf("languages %s/%s: %v", owner, repo, langErr)
		} else {
			item.Language = gh.PrimaryLanguage(langs)
		}
	}

	ownerProfile := repoMeta.Owner
	login := repoMeta.OwnerLogin()
	if login != "" {
		if cached, ok := owners.get(login); ok {
			ownerProfile = cached
		} else {
			fetched, fetchErr := fetchOwner(ctx, client, login, repoMeta.OwnerType())
			if fetchErr != nil && gh.IsRateLimited(fetchErr) {
				return item, fetchErr
			}
			if fetchErr != nil {
				log.Printf("owner %s: %v", login, fetchErr)
			} else {
				ownerProfile = fetched
				owners.put(fetched)
			}
		}
	}

	readme, readmeErr := client.GetReadme(ctx, owner, repo)
	if readmeErr != nil && gh.IsRateLimited(readmeErr) {
		return item, readmeErr
	}
	if readmeErr != nil {
		log.Printf("readme %s/%s: %v", owner, repo, readmeErr)
		readme = ""
	}

	packageHome := ""
	if strings.TrimSpace(repoMeta.Homepage) == "" {
		site, pkgErr := fetchPackageHomepage(ctx, client, owner, repo)
		if pkgErr != nil {
			if gh.IsRateLimited(pkgErr) {
				return item, pkgErr
			}
			log.Printf("package metadata %s/%s: %v", owner, repo, pkgErr)
		} else {
			packageHome = site
		}
	}

	ownerBlog := ""
	ownerType, location, company, bio := "", "", "", ""
	if ownerProfile != nil {
		ownerBlog = ownerProfile.Blog
		ownerType = ownerProfile.Type
		location = ownerProfile.Location
		company = ownerProfile.Company
		bio = ownerProfile.Bio
	}

	item.Website = resolveWebsite(repoMeta.Homepage, ownerBlog, readme, packageHome)

	rel, relErr := client.GetLatestRelease(ctx, owner, repo)
	if relErr != nil && gh.IsRateLimited(relErr) {
		return item, relErr
	}
	if relErr != nil {
		log.Printf("release %s/%s: %v", owner, repo, relErr)
	} else if rel != nil {
		item.LastRelease = formatTime(rel.PublishedAt)
		if item.LastRelease == "" {
			item.LastRelease = rel.TagName
		}
	}

	isOrg := strings.EqualFold(ownerType, "Organization") || strings.EqualFold(repoMeta.OwnerType(), "Organization")
	evidence, score := countryEvidence(profileSignals{
		OwnerType:   ownerType,
		Location:    location,
		Company:     company,
		Blog:        ownerBlog,
		Bio:         bio,
		Homepage:    item.Website,
		Readme:      readme,
		DocsWebsite: item.Website,
		IsOrg:       isOrg,
	}, matcher)
	if evidence == nil {
		evidence = []Evidence{}
	}
	if score < 40 {
		extra, extraScore, extraErr := developerEvidence(ctx, client, owners, matcher, owner, repo, log)
		if extraErr != nil {
			if gh.IsRateLimited(extraErr) {
				return item, extraErr
			}
			log.Printf("contributors %s/%s: %v", owner, repo, extraErr)
		} else {
			evidence = append(evidence, extra...)
			score += extraScore
			if score > 100 {
				score = 100
			}
		}
	}
	item.CountryEvidence = evidence
	item.CountryScore = score

	isFramework := looksLikeFramework(seed, item.Name, item.Description, readme, item.Topics)
	item.Status = resolveStatus(seed, true, isFramework, score)
	return item, nil
}

func fetchOwner(ctx context.Context, client *gh.Client, login, ownerType string) (*gh.User, error) {
	if strings.EqualFold(ownerType, "Organization") {
		org, err := client.GetOrganization(ctx, login)
		if err == nil {
			return org, nil
		}
		if gh.IsNotFound(err) {
			return client.GetUser(ctx, login)
		}
		return nil, err
	}
	u, err := client.GetUser(ctx, login)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(u.Type, "Organization") {
		if org, orgErr := client.GetOrganization(ctx, login); orgErr == nil {
			return org, nil
		}
	}
	return u, nil
}

func developerEvidence(ctx context.Context, client *gh.Client, owners *ownerCache, matcher *validation.CountryMatcher, owner, repo string, log Logger) ([]Evidence, int, error) {
	logins, err := client.ListContributorLogins(ctx, owner, repo, 3)
	if err != nil {
		return nil, 0, err
	}
	for _, login := range logins {
		if strings.EqualFold(login, owner) {
			continue
		}
		var user *gh.User
		if cached, ok := owners.get(login); ok {
			user = cached
		} else {
			fetched, fetchErr := client.GetUser(ctx, login)
			if fetchErr != nil {
				if gh.IsRateLimited(fetchErr) {
					return nil, 0, fetchErr
				}
				log.Printf("contributor %s: %v", login, fetchErr)
				continue
			}
			user = fetched
			owners.put(fetched)
		}
		if user == nil {
			continue
		}
		country, city := matcher.LocationMatches(user.Location)
		if !country && !city {
			continue
		}
		return []Evidence{{
			Type:   EvidenceDeveloperLocation,
			Value:  strings.TrimSpace(user.Location),
			Source: "github",
		}}, 30, nil
	}
	return nil, 0, nil
}

func fetchPackageHomepage(ctx context.Context, client *gh.Client, owner, repo string) (string, error) {
	for _, file := range []string{"package.json", "composer.json"} {
		raw, err := getRepoFile(ctx, client, owner, repo, file)
		if err != nil {
			if gh.IsNotFound(err) {
				continue
			}
			if gh.IsRateLimited(err) {
				return "", err
			}
			continue
		}
		if site := packageHomepageFromJSON(raw); site != "" {
			return site, nil
		}
	}
	return "", nil
}

func getRepoFile(ctx context.Context, client *gh.Client, owner, repo, path string) ([]byte, error) {
	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if err := client.GetJSON(ctx, apiPath, nil, &payload); err != nil {
		return nil, err
	}
	if payload.Content == "" {
		return nil, nil
	}
	if !strings.EqualFold(payload.Encoding, "base64") {
		return []byte(payload.Content), nil
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, payload.Content)
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func displayName(seed Seed, repoName string) string {
	if strings.TrimSpace(seed.Name) != "" {
		return strings.TrimSpace(seed.Name)
	}
	return repoName
}

func missingFramework(seed Seed, canonical, _ string, now time.Time) Framework {
	github := canonical
	if github == "" {
		github = seed.GitHub
	}
	return Framework{
		Name:            displayName(seed, github),
		Category:        seed.Category,
		GitHub:          github,
		License:         "Unknown",
		CountryEvidence: []Evidence{},
		Status:          StatusNotFound,
		ScannedAt:       now.UTC().Format(time.RFC3339),
	}
}

func countStatus(rep *Report, status string) {
	switch status {
	case StatusVerified:
		rep.Verified++
	case StatusPendingVerification:
		rep.PendingVerification++
	case StatusHistorical:
		rep.Historical++
	case StatusNotFound:
		rep.RepositoryNotFound++
	case StatusExcluded:
		rep.Excluded++
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
