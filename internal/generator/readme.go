package generator

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/framework"
	"github.com/serhankarakoc/turkish-open-source/internal/project"
)

const (
	GeneratedStart = "<!-- GENERATED:START -->"
	GeneratedEnd   = "<!-- GENERATED:END -->"
)

const staticHeader = `# 🇹🇷 Turkish Open Source

Türkiye'de geliştirilen açık kaynak framework ve projelerin otomatik güncellenen kataloğu.

[![Update](https://github.com/serhankarakoc/turkish-open-source/actions/workflows/update.yml/badge.svg)](https://github.com/serhankarakoc/turkish-open-source/actions/workflows/update.yml)
[![Validate](https://github.com/serhankarakoc/turkish-open-source/actions/workflows/validate.yml/badge.svg)](https://github.com/serhankarakoc/turkish-open-source/actions/workflows/validate.yml)
[![License: MIT](https://img.shields.io/github/license/serhankarakoc/turkish-open-source)](LICENSE)

Kaynak: [data/frameworks.json](data/frameworks.json) · [data/projects.json](data/projects.json) · [kriterler](docs/CRITERIA.md)

`

func Generate(projects []project.Project, frameworks []framework.Framework, cfg *config.Config, generatedAt time.Time) string {
	body := generatedBody(projects, frameworks, cfg, generatedAt)
	return staticHeader + GeneratedStart + "\n" + body + GeneratedEnd + "\n"
}

func Patch(existing string, projects []project.Project, frameworks []framework.Framework, cfg *config.Config, generatedAt time.Time) string {
	body := generatedBody(projects, frameworks, cfg, generatedAt)
	start := strings.Index(existing, GeneratedStart)
	end := strings.Index(existing, GeneratedEnd)
	if start >= 0 && end > start {
		end += len(GeneratedEnd)
		return existing[:start] + GeneratedStart + "\n" + body + GeneratedEnd + existing[end:]
	}
	if strings.TrimSpace(existing) == "" {
		return Generate(projects, frameworks, cfg, generatedAt)
	}
	trimmed := strings.TrimRight(existing, "\n") + "\n\n"
	return trimmed + GeneratedStart + "\n" + body + GeneratedEnd + "\n"
}

func WriteREADME(path string, existing string, projects []project.Project, frameworks []framework.Framework, cfg *config.Config, generatedAt time.Time) error {
	out := Patch(existing, projects, frameworks, cfg, generatedAt)
	if strings.HasPrefix(strings.TrimSpace(existing), "# 🇹🇷 Turkish Open Source") || strings.TrimSpace(existing) == "" {
		out = Generate(projects, frameworks, cfg, generatedAt)
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

func generatedBody(_ []project.Project, frameworks []framework.Framework, _ *config.Config, _ time.Time) string {
	visible := visibleFrameworks(frameworks)
	var b strings.Builder
	fmt.Fprintf(&b, "\n**%d** framework\n\n", len(visible))
	writeFrameworkTable(&b, visible)
	b.WriteString("## Katkı\n\n")
	b.WriteString("- Yeni proje: [Add Project](https://github.com/serhankarakoc/turkish-open-source/issues/new?template=add-project.yml)\n")
	b.WriteString("- Sorun bildir: [Report Project](https://github.com/serhankarakoc/turkish-open-source/issues/new?template=report-project.yml)\n\n")
	return b.String()
}

func writeFrameworkTable(b *strings.Builder, frameworks []framework.Framework) {
	b.WriteString("## Framework'ler\n\n")
	if len(frameworks) == 0 {
		b.WriteString("_Henüz listelenecek framework yok._\n\n")
		return
	}
	b.WriteString("| Framework | Website | Dil | Kategori | Stars | Lisans |\n")
	b.WriteString("|---|---|---|---|---:|---|\n")
	items := append([]framework.Framework(nil), frameworks...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Stars != items[j].Stars {
			return items[i].Stars > items[j].Stars
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	for _, f := range items {
		name := f.Name
		if name == "" {
			name = f.GitHubRepo
		}
		lang := dash(f.Language)
		license := dash(f.License)
		if license == "other" || license == "Unknown" {
			license = "-"
		}
		fmt.Fprintf(b, "| [%s](%s) | %s | %s | %s | %s | %s |\n",
			escapeTable(name), f.GitHub, frameworkWebsiteLink(f.Website), escapeTable(lang),
			escapeTable(frameworkCategoryLabel(f.Category)), strconv.Itoa(f.Stars), escapeTable(license))
	}
	b.WriteString("\n")
}

func visibleFrameworks(frameworks []framework.Framework) []framework.Framework {
	out := make([]framework.Framework, 0, len(frameworks))
	for _, f := range frameworks {
		switch f.Status {
		case framework.StatusNotFound, framework.StatusExcluded:
			continue
		}
		out = append(out, f)
	}
	return out
}

func frameworkCategoryLabel(key string) string {
	switch key {
	case "application-framework":
		return "Application"
	case "ui-framework":
		return "UI"
	case "web-framework":
		return "Web"
	case "micro-frontend-framework":
		return "Micro frontend"
	case "computer-vision-framework":
		return "Computer vision"
	case "testing-framework":
		return "Testing"
	default:
		if key == "" {
			return "-"
		}
		return key
	}
}

func frameworkWebsiteLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "-"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "-"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "-"
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if host == "" {
		return "-"
	}
	href := strings.TrimRight(raw, "/")
	return fmt.Sprintf("[%s](%s)", escapeTable(host), href)
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

func escapeTable(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
