package framework

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

var (
	websiteLabelURL = regexp.MustCompile(`(?i)(?:website|homepage|official site|docs|documentation)\s*[:\-]?\s*(?:\[([^\]]*)\]\()?<?(https?://[^\s)>]+)`)
	markdownURL     = regexp.MustCompile(`(?i)\[([^\]]*(?:website|homepage|docs|documentation)[^\]]*)\]\((https?://[^)\s]+)\)`)
)

var skipWebsiteHosts = []string{
	"github.com",
	"www.github.com",
	"raw.githubusercontent.com",
	"img.shields.io",
	"shields.io",
	"travis-ci.org",
	"travis-ci.com",
	"circleci.com",
	"codecov.io",
	"coveralls.io",
	"badge.fury.io",
	"gitter.im",
	"discord.com",
	"twitter.com",
	"x.com",
}

func resolveWebsite(homepage, ownerBlog, readme string, packageHomepage string) string {
	for _, candidate := range []string{homepage, packageHomepage, extractReadmeWebsite(readme), ownerBlog} {
		if site := cleanWebsite(candidate); site != "" {
			return site
		}
	}
	return ""
}

func extractReadmeWebsite(readme string) string {
	readme = strings.TrimSpace(readme)
	if readme == "" {
		return ""
	}
	if m := markdownURL.FindStringSubmatch(readme); len(m) == 3 {
		if site := cleanWebsite(m[2]); site != "" {
			return site
		}
	}
	if m := websiteLabelURL.FindStringSubmatch(readme); len(m) == 3 {
		if site := cleanWebsite(m[2]); site != "" {
			return site
		}
	}
	return ""
}

func packageHomepageFromJSON(raw []byte) string {
	var payload struct {
		Homepage string `json:"homepage"`
		Extra    struct {
			Homepage string `json:"homepage"`
		} `json:"extra"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if site := cleanWebsite(payload.Homepage); site != "" {
		return site
	}
	return cleanWebsite(payload.Extra.Homepage)
}

func cleanWebsite(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "<>\"'")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "mailto:") {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	for _, skip := range skipWebsiteHosts {
		if host == skip || strings.HasSuffix(host, "."+skip) {
			return ""
		}
	}
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}
