package validation

import "time"

type ActivityInput struct {
	PushedAt         time.Time
	UpdatedAt        time.Time
	HasRecentRelease bool
	ReleaseAt        time.Time
	ContributorCount int
	OpenIssues       int
	HasIssueActivity bool
	Now              time.Time
}

func ActivityScore(in ActivityInput) int {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	score := 0

	latest := in.PushedAt
	if in.UpdatedAt.After(latest) {
		latest = in.UpdatedAt
	}
	if !latest.IsZero() {
		days := now.Sub(latest).Hours() / 24
		switch {
		case days < 30:
			score += 40
		case days < 90:
			score += 25
		case days < 180:
			score += 15
		case days < 365:
			score += 8
		}
	}

	if in.HasRecentRelease && !in.ReleaseAt.IsZero() {
		days := now.Sub(in.ReleaseAt).Hours() / 24
		if days < 180 {
			score += 20
		} else {
			score += 8
		}
	}

	switch {
	case in.ContributorCount >= 5:
		score += 20
	case in.ContributorCount >= 2:
		score += 10
	}

	if in.HasIssueActivity {
		score += 10
	} else if in.OpenIssues > 0 {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func IsActive(archived bool, pushedAt, now time.Time, maxInactiveDays int) bool {
	if archived {
		return false
	}
	if maxInactiveDays <= 0 {
		maxInactiveDays = 365 * 3
	}
	if pushedAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Sub(pushedAt) <= time.Duration(maxInactiveDays)*24*time.Hour
}
