package discovery

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/generator"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
	"github.com/serhankarakoc/turkish-open-source/internal/project"
	"github.com/serhankarakoc/turkish-open-source/internal/scoring"
	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

type EvalStats struct {
	OpenSource  int
	Rejected    int
	Verified    int
	Likely      int
	NeedsReview int
	Excluded    int
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

func Evaluate(ctx context.Context, client *gh.Client, cfg *config.Config, candidates []*Candidate, now time.Time, log Logger) ([]project.Project, EvalStats, error) {
	if log == nil {
		log = nopLogger{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	workers := cfg.Scanner.MaxWorkers
	if workers <= 0 {
		workers = 6
	}
	if workers > 8 {
		workers = 8
	}

	jobs := make(chan *Candidate)
	var mu sync.Mutex
	var accepted []project.Project
	var stats EvalStats
	owners := newOwnerCache()
	matcher := validation.NewCountryMatcher(cfg)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for cand := range jobs {
				p, ok, band := evaluateOne(ctx, client, cfg, matcher, owners, cand, now, log)
				mu.Lock()
				if ok {
					accepted = append(accepted, p)
					stats.OpenSource++
				} else {
					stats.Rejected++
				}
				switch band {
				case scoring.StatusVerified:
					stats.Verified++
				case scoring.StatusLikely:
					stats.Likely++
				case scoring.StatusNeedsReview:
					stats.NeedsReview++
				case scoring.StatusExcluded:
					stats.Excluded++
				}
				mu.Unlock()
			}
		}()
	}

	for _, c := range candidates {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return accepted, stats, ctx.Err()
		case jobs <- c:
		}
	}
	close(jobs)
	wg.Wait()
	project.SortProjects(accepted)
	return accepted, stats, nil
}

func evaluateOne(ctx context.Context, client *gh.Client, cfg *config.Config, matcher *validation.CountryMatcher, owners *ownerCache, cand *Candidate, now time.Time, log Logger) (project.Project, bool, string) {
	repo := cand.Repository
	if repo.Private || repo.Disabled {
		return project.Project{}, false, ""
	}
	if repo.Fork && !cfg.Projects.IncludeForks {
		return project.Project{}, false, ""
	}
	if repo.Archived && !cfg.Projects.IncludeArchived {
		return project.Project{}, false, ""
	}

	licenseID := validation.NormalizeLicense(repo.LicenseID())
	licenseStatus := validation.LicenseStatus(licenseID)
	if cfg.Projects.RequireLicense && !cfg.Projects.ShowUnlicensed && licenseStatus != validation.LicenseValid {
		return project.Project{}, false, ""
	}

	owner := repo.Owner
	login := repo.OwnerLogin()
	if cfg.Enrichment.FetchOwner && login != "" && client != nil {
		if cached, ok := owners.get(login); ok {
			owner = cached
		} else {
			fetched, err := client.GetUser(ctx, login)
			if err != nil {
				log.Printf("owner %s: %v", login, err)
			} else {
				owner = fetched
				owners.put(fetched)
			}
		}
	}

	ownerLocation, ownerBio, ownerCompany, ownerBlog, ownerType := "", "", "", "", repo.OwnerType()
	if owner != nil {
		ownerLocation = owner.Location
		ownerBio = owner.Bio
		ownerCompany = owner.Company
		ownerBlog = owner.Blog
		if owner.Type != "" {
			ownerType = owner.Type
		}
	}

	topics := repo.Topics
	categories := generator.DetectCategories(topics, repo.Language, cfg.Categories)
	primaryCategory := "other"
	if len(categories) > 0 {
		primaryCategory = categories[0]
	}
	readme := ""
	needReadme := cfg.Enrichment.FetchReadme && client != nil
	if needReadme {
		country, city := matcher.LocationMatches(ownerLocation)
		if country || city || len(gh.TurkishTopics(topics)) > 0 || matcher.DescriptionSignal(repo.Description) || matcher.TextMentionsCountry(ownerBio) || matcher.TextMentionsCountry(ownerCompany) {
			text, err := client.GetReadme(ctx, repo.OwnerLogin(), repo.Name)
			if err != nil {
				log.Printf("readme %s: %v", repo.FullName, err)
			} else {
				readme = text
			}
		}
	}

	turkey := scoring.TurkeyScore(scoring.Input{
		OwnerType:         ownerType,
		OwnerLocation:     ownerLocation,
		OwnerBio:          ownerBio,
		OwnerCompany:      ownerCompany,
		OwnerBlog:         ownerBlog,
		Homepage:          repo.Homepage,
		Description:       repo.Description,
		Readme:            readme,
		Topics:            topics,
		OwnerOtherSignals: ownerHistorySignals(login, ownerLocation, topics, cfg),
		IsFork:            repo.Fork,
		IsArchived:        repo.Archived,
		IsMirror:          repo.MirrorURL != "",
		Category:          primaryCategory,
		Matcher:           matcher,
	})

	var releaseAt time.Time
	hasRelease := false
	if cfg.Enrichment.FetchRelease && client != nil && turkey.Score >= cfg.Projects.MinimumTurkeyScore-25 {
		rel, err := client.GetLatestRelease(ctx, repo.OwnerLogin(), repo.Name)
		if err != nil {
			log.Printf("release %s: %v", repo.FullName, err)
		} else if rel != nil {
			hasRelease = true
			releaseAt = rel.PublishedAt
		}
	}

	contributors := 0
	if cfg.Enrichment.FetchContributors && client != nil && turkey.Score >= cfg.Projects.MinimumTurkeyScore-25 {
		n, err := client.CountContributors(ctx, repo.OwnerLogin(), repo.Name)
		if err != nil {
			log.Printf("contributors %s: %v", repo.FullName, err)
		} else {
			contributors = n
		}
	}

	activity := validation.ActivityScore(validation.ActivityInput{
		PushedAt:         repo.PushedAt,
		UpdatedAt:        repo.UpdatedAt,
		HasRecentRelease: hasRelease,
		ReleaseAt:        releaseAt,
		ContributorCount: contributors,
		OpenIssues:       repo.OpenIssuesCount,
		HasIssueActivity: repo.OpenIssuesCount > 0 && now.Sub(repo.UpdatedAt) < 90*24*time.Hour,
		Now:              now,
	})
	quality := scoring.QualityScore(scoring.QualityInput{
		Stars:            repo.StargazersCount,
		Forks:            repo.ForksCount,
		Contributors:     contributors,
		OpenIssues:       repo.OpenIssuesCount,
		HasRecentRelease: hasRelease,
		ReleaseAt:        releaseAt,
		PushedAt:         repo.PushedAt,
		UpdatedAt:        repo.UpdatedAt,
		HasHomepage:      strings.TrimSpace(repo.Homepage) != "",
		HasDocs:          strings.TrimSpace(readme) != "",
		LicenseValid:     licenseStatus == validation.LicenseValid,
		Archived:         repo.Archived,
		Now:              now,
	})

	if turkey.Score < cfg.Projects.MinimumTurkeyScore {
		return project.Project{}, false, turkey.Status
	}
	if activity < cfg.Projects.MinimumActivityScore {
		return project.Project{}, false, turkey.Status
	}
	if quality < cfg.Projects.MinimumQualityScore {
		return project.Project{}, false, turkey.Status
	}
	if cfg.Projects.MinimumStars > 0 && repo.StargazersCount < cfg.Projects.MinimumStars {
		allowLowStar := cfg.Projects.AllowLowStarException &&
			turkey.Score >= cfg.Projects.LowStarExceptionTurkeyScore &&
			quality >= cfg.Projects.LowStarExceptionQualityScore &&
			isPriorityCategory(primaryCategory)
		if !allowLowStar {
			return project.Project{}, false, turkey.Status
		}
	}

	verified, verification := scoring.AutomatedVerification(turkey.Score, false)
	p := project.Project{
		ID:            repo.ID,
		Name:          repo.Name,
		FullName:      repo.FullName,
		Repository:    repo.FullName,
		URL:           repo.URL,
		HTMLURL:       repo.HTMLURL,
		Owner:         login,
		OwnerType:     ownerType,
		OwnerLocation: ownerLocation,
		Description:   repo.Description,
		Language:      repo.Language,
		License:       licenseID,
		LicenseStatus: licenseStatus,
		Stars:         repo.StargazersCount,
		Forks:         repo.ForksCount,
		OpenIssues:    repo.OpenIssuesCount,
		Topics:        project.CloneTopics(topics),
		Category:      primaryCategory,
		Categories:    categories,
		Homepage:      repo.Homepage,
		Country:       "TR",
		TurkeyScore:   turkey.Score,
		ActivityScore: activity,
		QualityScore:  quality,
		TurkeySignals: turkey.Signals,
		Evidence:      project.CloneStrings(turkey.Evidence),
		Source:        "github_scanner",
		Status:        turkey.Status,
		Verification:  verification,
		IsActive:      validation.IsActive(repo.Archived, repo.PushedAt, now, 365*3),
		IsArchived:    repo.Archived,
		IsFork:        repo.Fork,
		IsVerified:    verified,
		CreatedAt:     formatTime(repo.CreatedAt),
		UpdatedAt:     formatTime(repo.UpdatedAt),
		PushedAt:      formatTime(repo.PushedAt),
		LastScannedAt: now.UTC().Format(time.RFC3339),
	}
	return p, true, turkey.Status
}

func isPriorityCategory(category string) bool {
	switch category {
	case "framework", "library", "devtools", "api", "database", "devops", "networking", "security", "ai":
		return true
	default:
		return false
	}
}

func ownerHistorySignals(login, ownerLocation string, topics []string, cfg *config.Config) int {
	score := 0
	if login != "" && strings.Contains(strings.ToLower(login), "tr") {
		score++
	}
	if ownerLocation != "" {
		score++
	}
	priority := map[string]struct{}{}
	if cfg != nil {
		for _, topic := range cfg.Keywords.PriorityTopics {
			priority[gh.NormalizeTopic(topic)] = struct{}{}
		}
	}
	for _, topic := range topics {
		if _, ok := priority[gh.NormalizeTopic(topic)]; ok {
			score++
			break
		}
	}
	return score
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
