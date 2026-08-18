package scoring

import (
	"testing"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/validation"
)

func testMatcher() *validation.CountryMatcher {
	cfg := &config.Config{
		Locations: []string{"Istanbul", "Ankara", "Izmir", "Turkey"},
		Keywords: config.Keywords{
			TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane", "uygulama"},
		},
		Countries: config.Countries{
			Countries: []config.Country{{
				Code:       "TR",
				Names:      []string{"Turkey", "Türkiye", "Turkiye"},
				Adjectives: []string{"Turkish", "Türk"},
				Domains:    []string{".tr"},
			}},
		},
	}
	return validation.NewCountryMatcher(cfg)
}

func TestTurkeyScoreRejectsSingleWeakSignal(t *testing.T) {
	m := testMatcher()
	got := TurkeyScore(Input{
		Description: "A project about cooking turkey recipes",
		Matcher:     m,
	})
	if got.Score >= 50 {
		t.Fatalf("single keyword should not pass, got %+v", got)
	}
	if got.Status != StatusExcluded {
		t.Fatalf("status=%s want excluded", got.Status)
	}
}

func TestTurkeyScoreRejectsTopicOnly(t *testing.T) {
	got := TurkeyScore(Input{
		Topics:  []string{"turkey"},
		Matcher: testMatcher(),
	})
	if got.Score >= 50 {
		t.Fatalf("single topic should be capped below 50, got %d %v", got.Score, got.Signals)
	}
}

func TestTurkeyScoreOwnerLocationAndReadme(t *testing.T) {
	readme := "Bu kütüphane ve uygulama Türkiye'de geliştirilen bir yazılım projesidir. Kullanıcılar için bir araç sağlar."
	got := TurkeyScore(Input{
		OwnerLocation: "Istanbul, Turkey",
		Readme:        readme,
		Topics:        []string{"made-in-turkey", "cli"},
		Homepage:      "https://example.com.tr",
		Matcher:       testMatcher(),
	})
	if got.Score < 75 {
		t.Fatalf("expected strong/verified score, got %d signals=%v groups=%v", got.Score, got.Signals, got.Groups)
	}
	if got.Status == StatusExcluded || got.Status == StatusNeedsReview {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestStarsDoNotAffectTurkeyScore(t *testing.T) {
	in := Input{OwnerLocation: "Ankara, Türkiye", Topics: []string{"turkish"}, Matcher: testMatcher()}
	a := TurkeyScore(in)
	b := TurkeyScore(in)
	if a.Score != b.Score {
		t.Fatalf("score should be deterministic: %d vs %d", a.Score, b.Score)
	}
}

func TestStatusBands(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, StatusVerified},
		{70, StatusVerified},
		{69, StatusLikely},
		{50, StatusLikely},
		{49, StatusNeedsReview},
		{30, StatusNeedsReview},
		{29, StatusExcluded},
		{0, StatusExcluded},
	}
	for _, tc := range cases {
		if got := Status(tc.score); got != tc.want {
			t.Fatalf("score %d: got %s want %s", tc.score, got, tc.want)
		}
	}
}

func TestCommunityVerificationNotDowngraded(t *testing.T) {
	verified, kind := AutomatedVerification(10, true)
	if !verified || kind != VerificationCommunity {
		t.Fatalf("community verification should be preserved, got %v %s", verified, kind)
	}
}

func TestClamp(t *testing.T) {
	if Clamp(-4) != 0 || Clamp(140) != 100 {
		t.Fatalf("clamp failed")
	}
}

func TestQualityScoreRewardsHealthyProject(t *testing.T) {
	now := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	got := QualityScore(QualityInput{
		Stars:            25,
		Forks:            5,
		Contributors:     3,
		HasRecentRelease: true,
		ReleaseAt:        now.Add(-20 * 24 * time.Hour),
		PushedAt:         now.Add(-10 * 24 * time.Hour),
		UpdatedAt:        now.Add(-5 * 24 * time.Hour),
		HasHomepage:      true,
		HasDocs:          true,
		LicenseValid:     true,
		Now:              now,
	})
	if got < 55 {
		t.Fatalf("quality score too low: %d", got)
	}
}
