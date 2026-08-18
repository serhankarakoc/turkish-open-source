package discovery

import (
	"fmt"
	"strings"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
)

const (
	KindUserLocation = "user_location"
	KindOwnerPhrase  = "owner_phrase"
	KindRepoTopic    = "repo_topic"
	KindRepoLanguage = "repo_language"
	KindRepoKeyword  = "repo_keyword"
	KindRepoIntent   = "repo_intent"
)

type Query struct {
	Kind   string
	Source string
	Q      string
}

func BuildQueries(cfg *config.Config) []Query {
	if cfg == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []Query
	add := func(q Query) {
		key := q.Kind + "\t" + q.Q
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, q)
	}

	for _, loc := range cfg.Locations {
		loc = strings.TrimSpace(loc)
		if loc == "" {
			continue
		}
		quoted := quote(loc)
		add(Query{Kind: KindUserLocation, Source: "location:" + loc, Q: "location:" + quoted})
	}

	for _, phrase := range cfg.Keywords.SearchPhrases {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" {
			continue
		}
		add(Query{Kind: KindOwnerPhrase, Source: "owner-phrase:" + phrase, Q: quote(phrase)})
		add(Query{Kind: KindRepoKeyword, Source: "repo-phrase:" + phrase, Q: quote(phrase) + " in:name,description,readme"})
	}

	for _, topic := range cfg.Keywords.Topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		add(Query{Kind: KindRepoTopic, Source: "topic:" + topic, Q: "topic:" + topic})
	}

	countryLocs := cfg.Keywords.CountryLocations
	if len(countryLocs) == 0 {
		countryLocs = []string{"Turkey"}
	}
	for _, lang := range cfg.Keywords.Languages {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}
		add(Query{
			Kind:   KindRepoLanguage,
			Source: "language:" + lang,
			Q:      "language:" + quote(lang),
		})
	}

	for _, kw := range cfg.Keywords.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		add(Query{
			Kind:   KindRepoKeyword,
			Source: "keyword:" + kw,
			Q:      quote(kw) + " in:name,description,topics",
		})
	}

	for family, intents := range cfg.Keywords.IntentKeywords {
		for _, intent := range intents {
			intent = strings.TrimSpace(intent)
			if intent == "" {
				continue
			}
			add(Query{
				Kind:   KindRepoIntent,
				Source: "intent:" + family + ":" + intent,
				Q:      quote(intent) + " in:name,description,readme",
			})
			for _, loc := range countryLocs {
				add(Query{
					Kind:   KindRepoIntent,
					Source: "intent-location:" + family + ":" + intent + ":" + loc,
					Q:      quote(intent) + " " + quote(loc) + " in:name,description,readme",
				})
			}
			for _, lang := range cfg.Keywords.Languages {
				add(Query{
					Kind:   KindRepoIntent,
					Source: "intent-language:" + family + ":" + lang + ":" + intent,
					Q:      "language:" + quote(lang) + " " + quote(intent) + " in:name,description,readme",
				})
			}
		}
	}
	return out
}

func quote(v string) string {
	if strings.ContainsAny(v, " #:+") {
		return fmt.Sprintf("%q", v)
	}
	return v
}
