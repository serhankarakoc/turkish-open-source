package discovery

import (
	"testing"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
)

func TestBuildQueriesUsesConfig(t *testing.T) {
	cfg := &config.Config{
		Locations: []string{"Turkey", "Istanbul"},
		Keywords: config.Keywords{
			Topics:           []string{"turkey", "turkish"},
			CountryLocations: []string{"Turkey"},
			Languages:        []string{"Go", "Python"},
			Keywords:         []string{"türkiye"},
			SearchPhrases:    []string{"Turkish framework"},
			IntentKeywords:   map[string][]string{"framework": {"framework", "boilerplate"}},
		},
	}
	qs := BuildQueries(cfg)
	if len(qs) < 7 {
		t.Fatalf("expected multiple query strategies, got %d", len(qs))
	}
	kinds := map[string]int{}
	for _, q := range qs {
		kinds[q.Kind]++
		if q.Q == "" {
			t.Fatal("empty query")
		}
	}
	for _, kind := range []string{KindUserLocation, KindOwnerPhrase, KindRepoTopic, KindRepoLanguage, KindRepoKeyword, KindRepoIntent} {
		if kinds[kind] == 0 {
			t.Fatalf("missing kind %s", kind)
		}
	}
}

func TestBuildQueriesDedup(t *testing.T) {
	cfg := &config.Config{
		Locations: []string{"Turkey", "Turkey"},
		Keywords:  config.Keywords{Topics: []string{"turkey", "turkey"}},
	}
	qs := BuildQueries(cfg)
	seen := map[string]struct{}{}
	for _, q := range qs {
		key := q.Kind + q.Q
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate query %s", key)
		}
		seen[key] = struct{}{}
	}
}
