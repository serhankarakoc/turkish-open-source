package scoring

const (
	StatusVerified    = "verified"
	StatusLikely      = "likely"
	StatusNeedsReview = "needs_review"
	StatusExcluded    = "excluded"

	VerificationAutomated = "automated"
	VerificationCommunity = "community"
)

const (
	PointsOwnerLocation     = 30
	PointsOrganization      = 25
	PointsCityLocation      = 20
	PointsReadmeEvidence    = 20
	PointsWebsite           = 20
	PointsProfileText       = 15
	PointsOrganizationText  = 15
	PointsTurkishTopic      = 10
	PointsTurkishDocs       = 10
	PointsPackageCompany    = 10
	PointsMaintainerHistory = 10
	PointsMaintainers       = 10
	PointsDescription       = 10
	PointsMultiSignal       = 10
	PenaltyFork             = -30
	PenaltyMirror           = -30
	PenaltyArchived         = -20
	PenaltyForeignProject   = -20
	PenaltyTurkeyDataOnly   = -15
)

type Rule struct {
	ID     string
	Group  string
	Points int
	Reason string
}

func Rules() []Rule {
	return []Rule{
		{ID: "owner_location", Group: "geo", Points: PointsOwnerLocation, Reason: "GitHub owner profile location indicates Turkey"},
		{ID: "organization_profile", Group: "org", Points: PointsOrganization, Reason: "Organization profile indicates Turkey"},
		{ID: "owner_city", Group: "geo", Points: PointsCityLocation, Reason: "Owner location contains a Turkish city"},
		{ID: "readme_origin", Group: "readme", Points: PointsReadmeEvidence, Reason: "README indicates the project is developed in Turkey"},
		{ID: "project_website", Group: "website", Points: PointsWebsite, Reason: "Project website or homepage has a Turkey-linked domain or statement"},
		{ID: "owner_profile", Group: "profile", Points: PointsProfileText, Reason: "Owner bio or profile text indicates Turkey"},
		{ID: "organization_company", Group: "org", Points: PointsOrganizationText, Reason: "Organization/company text indicates Turkey"},
		{ID: "turkish_topic", Group: "topic", Points: PointsTurkishTopic, Reason: "Repository topics contain Turkey-linked terms"},
		{ID: "turkish_docs", Group: "readme", Points: PointsTurkishDocs, Reason: "Repository documentation is substantially in Turkish"},
		{ID: "package_company", Group: "website", Points: PointsPackageCompany, Reason: "Package or company links indicate a Turkey-based project"},
		{ID: "owner_project_history", Group: "maintainers", Points: PointsMaintainerHistory, Reason: "Owner's broader OSS footprint indicates Turkey"},
		{ID: "turkish_maintainers", Group: "maintainers", Points: PointsMaintainers, Reason: "Multiple maintainers signal Turkey"},
		{ID: "turkish_description", Group: "description", Points: PointsDescription, Reason: "Description contains Turkish language signals"},
		{ID: "multi_signal_bonus", Group: "bonus", Points: PointsMultiSignal, Reason: "Three or more independent Turkey signals"},
		{ID: "fork_penalty", Group: "penalty", Points: PenaltyFork, Reason: "Fork repositories are deprioritized"},
		{ID: "mirror_penalty", Group: "penalty", Points: PenaltyMirror, Reason: "Mirrors are excluded"},
		{ID: "archived_penalty", Group: "penalty", Points: PenaltyArchived, Reason: "Archived repositories are deprioritized"},
		{ID: "foreign_penalty", Group: "penalty", Points: PenaltyForeignProject, Reason: "Repository appears to be foreign-origin"},
		{ID: "data_only_penalty", Group: "penalty", Points: PenaltyTurkeyDataOnly, Reason: "Repository is about Turkey-related data, not necessarily Turkish-origin OSS"},
	}
}

func Status(score int) string {
	switch {
	case score >= 70:
		return StatusVerified
	case score >= 50:
		return StatusLikely
	case score >= 30:
		return StatusNeedsReview
	default:
		return StatusExcluded
	}
}

func Clamp(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
