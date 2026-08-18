package github

import "strings"

var turkishTopics = map[string]struct{}{
	"turkey":              {},
	"turkiye":             {},
	"türkiye":             {},
	"turkish":             {},
	"made-in-turkey":      {},
	"turkiye-yazilim":     {},
	"turkish-open-source": {},
}

func NormalizeTopic(topic string) string {
	t := strings.ToLower(strings.TrimSpace(topic))
	t = strings.ReplaceAll(t, "ı", "i")
	t = strings.ReplaceAll(t, "İ", "i")
	t = strings.ReplaceAll(t, "ü", "u")
	t = strings.ReplaceAll(t, "ö", "o")
	t = strings.ReplaceAll(t, "ş", "s")
	t = strings.ReplaceAll(t, "ç", "c")
	t = strings.ReplaceAll(t, "ğ", "g")
	return t
}

func IsTurkishTopic(topic string) bool {
	n := NormalizeTopic(topic)
	_, ok := turkishTopics[n]
	if ok {
		return true
	}
	_, ok = turkishTopics[strings.ToLower(strings.TrimSpace(topic))]
	return ok
}

func TurkishTopics(topics []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, t := range topics {
		if !IsTurkishTopic(t) {
			continue
		}
		key := NormalizeTopic(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.ToLower(strings.TrimSpace(t)))
	}
	return out
}
