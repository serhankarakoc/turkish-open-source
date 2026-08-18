package generator

import (
	"sort"
	"strings"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/project"
)

type CategoryStat struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Emoji  string `json:"emoji,omitempty"`
	Count  int    `json:"count"`
	Active int    `json:"active"`
	Stars  int    `json:"stars"`
}

type CategoryFile struct {
	Version     int            `json:"version"`
	GeneratedAt string         `json:"generated_at"`
	Categories  []CategoryStat `json:"categories"`
}

var languageCategory = map[string]string{
	"Swift":       "mobile",
	"Kotlin":      "mobile",
	"Dart":        "mobile",
	"Objective-C": "mobile",
	"HTML":        "web",
	"CSS":         "web",
	"Vue":         "web",
}

func DetectCategory(topics []string, language string, categories map[string]config.Category) string {
	all := DetectCategories(topics, language, categories)
	if len(all) == 0 {
		return "other"
	}
	return all[0]
}

func DetectCategories(topics []string, language string, categories map[string]config.Category) []string {
	scores := map[string]int{}
	for _, topic := range topics {
		t := strings.ToLower(strings.TrimSpace(topic))
		if t == "" {
			continue
		}
		for key, cat := range categories {
			if key == "other" {
				continue
			}
			for _, candidate := range cat.Topics {
				if t == strings.ToLower(candidate) {
					scores[key]++
				}
			}
		}
	}
	bestKey := ""
	bestScore := 0
	var out []string
	for key, score := range scores {
		if score > bestScore || (score == bestScore && (bestKey == "" || key < bestKey)) {
			bestScore = score
			bestKey = key
		}
	}
	if bestScore > 0 {
		for key, score := range scores {
			if score > 0 {
				out = append(out, key)
			}
		}
		sort.Strings(out)
		sort.SliceStable(out, func(i, j int) bool {
			if scores[out[i]] != scores[out[j]] {
				return scores[out[i]] > scores[out[j]]
			}
			return out[i] < out[j]
		})
		return out
	}
	if mapped, ok := languageCategory[language]; ok {
		if _, exists := categories[mapped]; exists {
			return []string{mapped}
		}
	}
	return []string{"other"}
}

func BuildCategoryFile(projects []project.Project, cfg *config.Config, generatedAt string) CategoryFile {
	stats := map[string]*CategoryStat{}
	if cfg != nil {
		for key, cat := range cfg.Categories {
			stats[key] = &CategoryStat{Key: key, Label: cat.Label, Emoji: cat.Emoji}
		}
	}
	for _, p := range projects {
		st, ok := stats[p.Category]
		if !ok {
			st = &CategoryStat{Key: p.Category, Label: p.Category}
			stats[p.Category] = st
		}
		st.Count++
		st.Stars += p.Stars
		if p.IsActive && !p.IsArchived {
			st.Active++
		}
	}
	out := make([]CategoryStat, 0, len(stats))
	for _, st := range stats {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return CategoryFile{
		Version:     project.DatasetVersion,
		GeneratedAt: generatedAt,
		Categories:  out,
	}
}
