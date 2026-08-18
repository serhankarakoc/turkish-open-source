package validation

import (
	"testing"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
)

func TestFoldTurkish(t *testing.T) {
	if Fold("Türkiye") != Fold("Turkiye") {
		t.Fatalf("Türkiye should fold to the same value as Turkiye: %q vs %q", Fold("Türkiye"), Fold("Turkiye"))
	}
	if Fold("İSTANBUL") == "" {
		t.Fatal("istanbul fold empty")
	}
}

func TestLocationMatchesCountryAndCity(t *testing.T) {
	m := NewCountryMatcher(&config.Config{
		Locations: []string{"Istanbul", "Ankara", "Turkey"},
		Countries: config.Countries{Countries: []config.Country{{
			Names:   []string{"Turkey", "Türkiye"},
			Domains: []string{".tr"},
		}}},
	})
	country, city := m.LocationMatches("Istanbul, Turkey")
	if !country || !city {
		t.Fatalf("country=%v city=%v", country, city)
	}
	country, city = m.LocationMatches("Berlin, Germany")
	if country || city {
		t.Fatalf("germany should not match, country=%v city=%v", country, city)
	}
}

func TestWebsiteMatchesTR(t *testing.T) {
	m := NewCountryMatcher(&config.Config{
		Countries: config.Countries{Countries: []config.Country{{Domains: []string{".tr"}}}},
	})
	if !m.WebsiteMatches("https://www.example.com.tr/app") {
		t.Fatal("expected .tr domain to match")
	}
	if m.WebsiteMatches("https://example.com") {
		t.Fatal("generic domain should not match")
	}
}

func TestReadmeSignalRequiresSubstance(t *testing.T) {
	m := NewCountryMatcher(&config.Config{
		Keywords: config.Keywords{TurkishStopwords: []string{"ve", "bir", "için", "yazılım", "kütüphane"}},
	})
	if m.ReadmeSignal("turkey").Strong {
		t.Fatal("one latin keyword must not be a strong README signal")
	}
	text := "Bu kütüphane ve yazılım projesi kullanıcılar için geliştirildi."
	sig := m.ReadmeSignal(text)
	if !sig.Strong && !sig.Weak {
		t.Fatalf("expected Turkish README signal, got %+v", sig)
	}
}
