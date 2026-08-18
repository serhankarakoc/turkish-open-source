package validation

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
)

var turkishLetters = regexp.MustCompile(`[ğüşıöçĞÜŞİÖÇ]`)

type CountryMatcher struct {
	names      []string
	adjectives []string
	domains    []string
	cities     []string
	stopwords  []string
}

func NewCountryMatcher(cfg *config.Config) *CountryMatcher {
	m := &CountryMatcher{}
	if cfg == nil {
		return m
	}
	for _, c := range cfg.Countries.Countries {
		m.names = append(m.names, c.Names...)
		m.adjectives = append(m.adjectives, c.Adjectives...)
		m.domains = append(m.domains, c.Domains...)
	}
	m.cities = append(m.cities, cfg.Locations...)
	m.stopwords = append(m.stopwords, cfg.Keywords.TurkishStopwords...)
	return m
}

func Fold(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ToLowerSpecial(unicode.TurkishCase, s)
	replacer := strings.NewReplacer(
		"ı", "i",
		"i̇", "i",
		"ğ", "g",
		"ü", "u",
		"ş", "s",
		"ö", "o",
		"ç", "c",
		"â", "a",
		"î", "i",
		"û", "u",
	)
	return replacer.Replace(s)
}

func containsFold(haystack, needle string) bool {
	h := Fold(haystack)
	n := Fold(needle)
	if h == "" || n == "" {
		return false
	}
	return strings.Contains(h, n)
}

func (m *CountryMatcher) LocationMatches(location string) (country bool, city bool) {
	if strings.TrimSpace(location) == "" {
		return false, false
	}
	for _, name := range m.names {
		if containsFold(location, name) {
			country = true
			break
		}
	}
	for _, cityName := range m.cities {
		if Fold(cityName) == Fold("Turkey") || Fold(cityName) == Fold("Türkiye") || Fold(cityName) == Fold("Turkiye") {
			continue
		}
		if containsFold(location, cityName) {
			city = true
			break
		}
	}
	return country, city
}

func (m *CountryMatcher) TextMentionsCountry(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	for _, name := range m.names {
		if containsFold(text, name) {
			return true
		}
	}
	for _, adj := range m.adjectives {
		if containsFold(text, adj) {
			return true
		}
	}
	return false
}

func (m *CountryMatcher) WebsiteMatches(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, d := range m.domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if strings.HasSuffix(host, d) {
			return true
		}
	}
	return false
}

type ReadmeSignal struct {
	Strong bool
	Weak   bool
}

func (m *CountryMatcher) ReadmeSignal(text string) ReadmeSignal {
	text = strings.TrimSpace(text)
	if text == "" {
		return ReadmeSignal{}
	}
	letters := len(turkishLetters.FindAllString(text, 40))
	stops := 0
	folded := Fold(text)
	fields := strings.Fields(folded)
	wordSet := map[string]int{}
	for _, w := range fields {
		w = strings.Trim(w, ".,;:!?()[]{}\"'`")
		if w == "" {
			continue
		}
		wordSet[w]++
	}
	for _, stop := range m.stopwords {
		stop = Fold(stop)
		if stop == "" {
			continue
		}
		stops += wordSet[stop]
	}
	strong := letters >= 8 || stops >= 6
	weak := !strong && (letters >= 3 || stops >= 3)
	return ReadmeSignal{Strong: strong, Weak: weak}
}

func (m *CountryMatcher) DescriptionSignal(text string) bool {
	sig := m.ReadmeSignal(text)
	return sig.Strong || sig.Weak
}
