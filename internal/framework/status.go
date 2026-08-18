package framework

import (
	"strings"
)

var frameworkPhrases = []string{
	"framework",
	"web framework",
	"application framework",
	"backend framework",
	"frontend framework",
	"ui framework",
	"micro frontend",
	"micro-frontend",
	"testing framework",
	"ml framework",
	"computer vision framework",
	"application development framework",
}

var antiFrameworkPhrases = []string{
	"api client",
	"sdk for",
	"command-line interface",
	"cli tool",
	"starter template",
	"starter kit",
	"this application is",
	"mobile app",
	"chrome extension",
}

func looksLikeFramework(seed Seed, name, description, readme string, topics []string) bool {
	category := strings.ToLower(seed.Category)
	if strings.Contains(category, "framework") {
		return true
	}
	blob := strings.ToLower(strings.Join([]string{name, description, strings.Join(topics, " "), truncate(readme, 8000)}, " "))
	hits := 0
	for _, phrase := range frameworkPhrases {
		if strings.Contains(blob, phrase) {
			hits++
		}
	}
	anti := 0
	for _, phrase := range antiFrameworkPhrases {
		if strings.Contains(blob, phrase) {
			anti++
		}
	}
	if hits == 0 && anti > 0 {
		return false
	}
	return hits > 0
}

func resolveStatus(seed Seed, found, isFramework bool, countryScore int) string {
	if !found {
		return StatusNotFound
	}
	if !isFramework {
		return StatusExcluded
	}
	if seed.InitialStatus == StatusHistorical {
		return StatusHistorical
	}
	if countryScore >= 40 {
		return StatusVerified
	}
	if seed.InitialStatus == StatusVerified && countryScore >= 30 {
		return StatusVerified
	}
	return StatusPendingVerification
}
