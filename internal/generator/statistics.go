package generator

import (
	"sort"

	"github.com/serhankarakoc/turkish-open-source/internal/project"
)

type Statistics struct {
	TotalProjects       int            `json:"total_projects"`
	ActiveProjects      int            `json:"active_projects"`
	ArchivedProjects    int            `json:"archived_projects"`
	VerifiedProjects    int            `json:"verified_projects"`
	LikelyProjects      int            `json:"likely_projects"`
	NeedsReviewProjects int            `json:"needs_review_projects"`
	TotalStars          int            `json:"total_stars"`
	TotalForks          int            `json:"total_forks"`
	Languages           int            `json:"languages"`
	Categories          int            `json:"categories"`
	LanguageCounts      map[string]int `json:"language_counts"`
	CategoryCounts      map[string]int `json:"category_counts"`
}

func ComputeStatistics(projects []project.Project) Statistics {
	st := Statistics{
		LanguageCounts: map[string]int{},
		CategoryCounts: map[string]int{},
	}
	for _, p := range projects {
		st.TotalProjects++
		st.TotalStars += p.Stars
		st.TotalForks += p.Forks
		if p.IsArchived {
			st.ArchivedProjects++
		}
		if p.IsActive && !p.IsArchived {
			st.ActiveProjects++
		}
		if p.IsVerified {
			st.VerifiedProjects++
		}
		switch p.Status {
		case "likely":
			st.LikelyProjects++
		case "needs_review":
			st.NeedsReviewProjects++
		}
		lang := p.Language
		if lang == "" {
			lang = "Unknown"
		}
		st.LanguageCounts[lang]++
		cat := p.Category
		if cat == "" {
			cat = "other"
		}
		st.CategoryCounts[cat]++
	}
	st.Languages = len(st.LanguageCounts)
	st.Categories = len(st.CategoryCounts)
	return st
}

func TopLanguages(counts map[string]int, limit int) []string {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for name, count := range counts {
		pairs = append(pairs, pair{name, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].name < pairs[j].name
	})
	if limit > 0 && len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]string, len(pairs))
	for i, p := range pairs {
		out[i] = p.name
	}
	return out
}
