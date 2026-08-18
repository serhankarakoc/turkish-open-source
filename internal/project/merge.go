package project

import "strings"

func Merge(existing, incoming Project) Project {
	out := incoming

	if existing.FirstDiscoveredAt != "" {
		out.FirstDiscoveredAt = existing.FirstDiscoveredAt
	}
	if existing.ID != 0 {
		out.StarsPrevious = existing.Stars
		out.StarDelta = incoming.Stars - existing.Stars
	}

	if IsCommunityVerified(existing) {
		out.IsVerified = true
		out.Verification = "community"
	}

	if existing.ManualCategory && existing.Category != "" {
		out.Category = existing.Category
		out.ManualCategory = true
	}
	if existing.Source == "manual" && out.Source == "" {
		out.Source = existing.Source
	}
	if len(existing.Categories) > 0 && len(out.Categories) == 0 {
		out.Categories = append([]string(nil), existing.Categories...)
	}

	if len(out.Topics) == 0 {
		out.Topics = []string{}
	}
	if len(out.Categories) == 0 && out.Category != "" {
		out.Categories = []string{out.Category}
	}
	return out
}

func MergeAll(existing []Project, incoming []Project, keepMissingVerified bool) []Project {
	prev := IndexByID(existing)
	seen := map[int64]struct{}{}
	out := make([]Project, 0, len(incoming)+len(existing))

	for _, p := range incoming {
		if old, ok := prev[p.ID]; ok {
			p = Merge(old, p)
		} else if p.FirstDiscoveredAt == "" {
			p.FirstDiscoveredAt = p.LastScannedAt
		}
		out = append(out, p)
		seen[p.ID] = struct{}{}
	}

	if keepMissingVerified {
		for _, old := range existing {
			if _, ok := seen[old.ID]; ok {
				continue
			}
			if IsCommunityVerified(old) {
				out = append(out, old)
			}
		}
	}

	byName := map[string]int{}
	deduped := make([]Project, 0, len(out))
	for _, p := range out {
		key := strings.ToLower(p.FullName)
		if i, ok := byName[key]; ok {
			if IsCommunityVerified(p) && !IsCommunityVerified(deduped[i]) {
				deduped[i] = p
			}
			continue
		}
		byName[key] = len(deduped)
		deduped = append(deduped, p)
	}

	SortProjects(deduped)
	return deduped
}
