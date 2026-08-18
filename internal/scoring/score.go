package scoring

import (
	"sort"

	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

type Input struct {
	OwnerType          string
	OwnerLocation      string
	OwnerBio           string
	OwnerCompany       string
	OwnerBlog          string
	Homepage           string
	Description        string
	Readme             string
	Topics             []string
	TurkishMaintainers int
	OwnerOtherSignals  int
	IsFork             bool
	IsArchived         bool
	IsMirror           bool
	Category           string
	Matcher            *validation.CountryMatcher
}

type Result struct {
	Score    int
	Status   string
	Signals  []string
	Groups   []string
	Evidence []string
}

func TurkeyScore(in Input) Result {
	matcher := in.Matcher
	if matcher == nil {
		matcher = validation.NewCountryMatcher(nil)
	}

	type hit struct {
		id       string
		group    string
		points   int
		evidence string
	}
	var hits []hit

	country, city := matcher.LocationMatches(in.OwnerLocation)
	if country {
		points := PointsOwnerLocation
		if in.OwnerType == "Organization" {
			points = PointsOrganization
		}
		hits = append(hits, hit{id: "owner_location", group: "geo", points: points, evidence: "GitHub owner profile location: Turkey"})
		if city {
			hits = append(hits, hit{id: "owner_city_bonus", group: "geo", points: 5, evidence: "GitHub owner profile location includes a Turkish city"})
		}
	} else if city {
		hits = append(hits, hit{id: "owner_city", group: "geo", points: PointsCityLocation, evidence: "GitHub owner profile location includes a Turkish city"})
	}

	readme := matcher.ReadmeSignal(in.Readme)
	if readme.Strong {
		hits = append(hits, hit{id: "readme_origin", group: "readme", points: PointsReadmeEvidence, evidence: "Repository README contains strong Turkish-origin signals"})
	} else if readme.Weak {
		hits = append(hits, hit{id: "turkish_docs", group: "readme", points: PointsTurkishDocs, evidence: "Repository documentation contains Turkish-language signals"})
	}

	if len(gh.TurkishTopics(in.Topics)) > 0 {
		hits = append(hits, hit{id: "turkish_topic", group: "topic", points: PointsTurkishTopic, evidence: "Repository topics include Turkey-linked terms"})
	}

	if matcher.WebsiteMatches(in.Homepage) || matcher.WebsiteMatches(in.OwnerBlog) {
		hits = append(hits, hit{id: "project_website", group: "website", points: PointsWebsite, evidence: "Project website or owner blog is linked to a .tr domain"})
	}

	if in.TurkishMaintainers >= 2 {
		hits = append(hits, hit{id: "turkish_maintainers", group: "maintainers", points: PointsMaintainers, evidence: "Multiple maintainers carry Turkey-linked signals"})
	}

	if matcher.DescriptionSignal(in.Description) {
		hits = append(hits, hit{id: "turkish_description", group: "description", points: PointsDescription, evidence: "Repository description contains Turkish-language signals"})
	}

	if matcher.TextMentionsCountry(in.OwnerBio) || matcher.TextMentionsCountry(in.OwnerCompany) {
		hits = append(hits, hit{id: "owner_profile", group: "profile", points: PointsProfileText, evidence: "Owner bio or company mentions Turkey"})
	}

	if in.OwnerOtherSignals > 0 {
		hits = append(hits, hit{id: "owner_project_history", group: "maintainers", points: min(PointsMaintainerHistory, in.OwnerOtherSignals*5), evidence: "Owner's other public OSS footprint suggests a Turkey-based maintainer"})
	}

	if in.IsFork {
		hits = append(hits, hit{id: "fork_penalty", group: "penalty", points: PenaltyFork, evidence: "Fork repository"})
	}
	if in.IsMirror {
		hits = append(hits, hit{id: "mirror_penalty", group: "penalty", points: PenaltyMirror, evidence: "Mirror repository"})
	}
	if in.IsArchived {
		hits = append(hits, hit{id: "archived_penalty", group: "penalty", points: PenaltyArchived, evidence: "Archived repository"})
	}
	if in.Category == "data" && !country && len(gh.TurkishTopics(in.Topics)) > 0 && !readme.Strong {
		hits = append(hits, hit{id: "data_only_penalty", group: "penalty", points: PenaltyTurkeyDataOnly, evidence: "Repository appears to focus on Turkey-related data rather than Turkish-origin software"})
	}

	groups := map[string]struct{}{}
	score := 0
	signals := make([]string, 0, len(hits))
	evidence := make([]string, 0, len(hits))
	for _, h := range hits {
		score += h.points
		signals = append(signals, h.id)
		if h.evidence != "" {
			evidence = append(evidence, h.evidence)
		}
		if h.group != "bonus" && h.group != "penalty" {
			groups[h.group] = struct{}{}
		}
	}

	if len(groups) >= 3 {
		score += PointsMultiSignal
		signals = append(signals, "multi_signal_bonus")
		evidence = append(evidence, "Multiple independent Turkey-linked signals align")
		groups["bonus"] = struct{}{}
	}

	score = Clamp(score)
	if len(groups) < 2 {
		if score > 49 {
			score = 49
		}
	}

	groupList := make([]string, 0, len(groups))
	for g := range groups {
		groupList = append(groupList, g)
	}
	sort.Strings(groupList)
	sort.Strings(signals)

	return Result{
		Score:    score,
		Status:   Status(score),
		Signals:  signals,
		Groups:   groupList,
		Evidence: evidence,
	}
}

func AutomatedVerification(score int, community bool) (verified bool, verification string) {
	if community {
		return true, VerificationCommunity
	}
	if score >= 90 {
		return true, VerificationAutomated
	}
	return false, VerificationAutomated
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
