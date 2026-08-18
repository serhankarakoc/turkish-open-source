package scoring

import "time"

type QualityInput struct {
	Stars            int
	Forks            int
	Contributors     int
	OpenIssues       int
	HasRecentRelease bool
	ReleaseAt        time.Time
	PushedAt         time.Time
	UpdatedAt        time.Time
	HasHomepage      bool
	HasDocs          bool
	LicenseValid     bool
	Archived         bool
	Now              time.Time
}

func QualityScore(in QualityInput) int {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	score := 0

	switch {
	case in.Stars >= 500:
		score += 30
	case in.Stars >= 100:
		score += 24
	case in.Stars >= 30:
		score += 16
	case in.Stars >= 10:
		score += 10
	case in.Stars >= 3:
		score += 5
	}

	switch {
	case in.Forks >= 100:
		score += 12
	case in.Forks >= 25:
		score += 8
	case in.Forks >= 5:
		score += 4
	case in.Forks >= 1:
		score += 2
	}

	if in.Contributors >= 5 {
		score += 10
	} else if in.Contributors >= 2 {
		score += 6
	}

	latest := in.PushedAt
	if in.UpdatedAt.After(latest) {
		latest = in.UpdatedAt
	}
	if !latest.IsZero() {
		days := now.Sub(latest).Hours() / 24
		switch {
		case days < 30:
			score += 18
		case days < 90:
			score += 14
		case days < 180:
			score += 10
		case days < 365:
			score += 6
		}
	}

	if in.HasRecentRelease && !in.ReleaseAt.IsZero() {
		if now.Sub(in.ReleaseAt) < 365*24*time.Hour {
			score += 10
		} else {
			score += 4
		}
	}

	if in.HasHomepage {
		score += 5
	}
	if in.HasDocs {
		score += 5
	}
	if in.LicenseValid {
		score += 5
	}
	if in.OpenIssues > 0 {
		score += 2
	}
	if in.Archived {
		score -= 20
	}
	return Clamp(score)
}
