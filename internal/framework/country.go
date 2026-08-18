package framework

import (
	"strings"

	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

const (
	EvidenceOrganizationLocation = "organization_location"
	EvidenceOwnerLocation        = "owner_location"
	EvidenceDeveloperLocation    = "developer_location"
	EvidenceCompanyLocation      = "company_location"
	EvidenceOfficialWebsite      = "official_website"
	EvidenceOfficialDocs         = "official_documentation"
	EvidenceReadme               = "repository_readme"
	EvidenceOrgProfile           = "organization_profile"
)

const (
	pointsOrganizationLocation = 40
	pointsOwnerLocation        = 30
	pointsOfficialWebsite      = 20
	pointsReadme               = 20
	pointsOfficialDocs         = 20
	pointsCompanyLocation      = 20
	pointsOrgProfile           = 15
)

type profileSignals struct {
	OwnerType   string
	Location    string
	Company     string
	Blog        string
	Bio         string
	Homepage    string
	Readme      string
	DocsWebsite string
	IsOrg       bool
}

func countryEvidence(p profileSignals, matcher *validation.CountryMatcher) ([]Evidence, int) {
	if matcher == nil {
		matcher = validation.NewCountryMatcher(nil)
	}
	var evidence []Evidence
	score := 0
	add := func(typ, value string, points int) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		evidence = append(evidence, Evidence{Type: typ, Value: value, Source: "github"})
		score += points
	}

	country, city := matcher.LocationMatches(p.Location)
	if country || city {
		label := strings.TrimSpace(p.Location)
		if p.IsOrg || strings.EqualFold(p.OwnerType, "Organization") {
			add(EvidenceOrganizationLocation, label, pointsOrganizationLocation)
		} else {
			add(EvidenceOwnerLocation, label, pointsOwnerLocation)
		}
	}

	if matcher.TextMentionsCountry(p.Company) {
		add(EvidenceCompanyLocation, strings.TrimSpace(p.Company), pointsCompanyLocation)
	}

	if matcher.WebsiteMatches(p.Homepage) || matcher.WebsiteMatches(p.Blog) {
		site := strings.TrimSpace(p.Homepage)
		if site == "" {
			site = strings.TrimSpace(p.Blog)
		}
		add(EvidenceOfficialWebsite, site, pointsOfficialWebsite)
	}

	readme := matcher.ReadmeSignal(p.Readme)
	if readme.Strong || (readme.Weak && matcher.TextMentionsCountry(p.Readme)) {
		add(EvidenceReadme, "README contains Turkey-origin language or company signals", pointsReadme)
	}

	if p.DocsWebsite != "" && (matcher.WebsiteMatches(p.DocsWebsite) || matcher.TextMentionsCountry(p.DocsWebsite)) {
		add(EvidenceOfficialDocs, p.DocsWebsite, pointsOfficialDocs)
	}

	if p.IsOrg || strings.EqualFold(p.OwnerType, "Organization") {
		if matcher.TextMentionsCountry(p.Bio) || matcher.TextMentionsCountry(p.Blog) {
			val := strings.TrimSpace(p.Bio)
			if val == "" {
				val = strings.TrimSpace(p.Blog)
			}
			add(EvidenceOrgProfile, val, pointsOrgProfile)
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return evidence, score
}
