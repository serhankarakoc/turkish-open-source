package project

import (
	"fmt"
	"net/url"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

func ValidateDataset(ds Dataset, allowedCategories map[string]struct{}) []error {
	var errs []error
	if ds.Version != DatasetVersion {
		errs = append(errs, ValidationError{Field: "version", Message: fmt.Sprintf("expected %d, got %d", DatasetVersion, ds.Version)})
	}
	if ds.Projects == nil {
		errs = append(errs, ValidationError{Field: "projects", Message: "must be an array"})
		return errs
	}

	ids := map[int64]string{}
	names := map[string]int64{}
	for i, p := range ds.Projects {
		prefix := fmt.Sprintf("projects[%d]", i)
		if p.ID <= 0 {
			errs = append(errs, ValidationError{Field: prefix + ".id", Message: "must be a positive GitHub repository id"})
		} else if prev, ok := ids[p.ID]; ok {
			errs = append(errs, ValidationError{Field: prefix + ".id", Message: fmt.Sprintf("duplicate repository id %d (also %s)", p.ID, prev)})
		} else {
			ids[p.ID] = p.FullName
		}

		if strings.TrimSpace(p.FullName) == "" || !strings.Contains(p.FullName, "/") {
			errs = append(errs, ValidationError{Field: prefix + ".full_name", Message: "must be owner/name"})
		} else {
			key := strings.ToLower(p.FullName)
			if prev, ok := names[key]; ok {
				errs = append(errs, ValidationError{Field: prefix + ".full_name", Message: fmt.Sprintf("duplicate repository %s (ids %d and %d)", p.FullName, prev, p.ID)})
			} else {
				names[key] = p.ID
			}
		}

		if p.Name == "" {
			errs = append(errs, ValidationError{Field: prefix + ".name", Message: "required"})
		}
		if p.Owner == "" {
			errs = append(errs, ValidationError{Field: prefix + ".owner", Message: "required"})
		}
		if err := validateURL(p.HTMLURL); err != nil {
			errs = append(errs, ValidationError{Field: prefix + ".html_url", Message: err.Error()})
		}
		if p.URL != "" {
			if err := validateURL(p.URL); err != nil {
				errs = append(errs, ValidationError{Field: prefix + ".url", Message: err.Error()})
			}
		}
		if p.Category == "" {
			errs = append(errs, ValidationError{Field: prefix + ".category", Message: "required"})
		} else if allowedCategories != nil {
			if _, ok := allowedCategories[p.Category]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".category", Message: "unknown category " + p.Category})
			}
		}
		for j, cat := range p.Categories {
			if allowedCategories != nil {
				if _, ok := allowedCategories[cat]; !ok {
					errs = append(errs, ValidationError{Field: fmt.Sprintf("%s.categories[%d]", prefix, j), Message: "unknown category " + cat})
				}
			}
		}
		if p.TurkeyScore < 0 || p.TurkeyScore > 100 {
			errs = append(errs, ValidationError{Field: prefix + ".turkey_score", Message: "must be 0-100"})
		}
		if p.ActivityScore < 0 || p.ActivityScore > 100 {
			errs = append(errs, ValidationError{Field: prefix + ".activity_score", Message: "must be 0-100"})
		}
		if p.QualityScore < 0 || p.QualityScore > 100 {
			errs = append(errs, ValidationError{Field: prefix + ".quality_score", Message: "must be 0-100"})
		}
		if p.Status != "" {
			switch p.Status {
			case "verified", "likely", "needs_review", "excluded":
			default:
				errs = append(errs, ValidationError{Field: prefix + ".status", Message: "must be verified, likely, needs_review, or excluded"})
			}
		}
		if p.LicenseStatus != "" {
			switch p.LicenseStatus {
			case "valid", "unknown", "unsupported":
			default:
				errs = append(errs, ValidationError{Field: prefix + ".license_status", Message: "must be valid, unknown, or unsupported"})
			}
		}
		if p.License != "" && strings.ContainsAny(p.License, " \t\n") {
			errs = append(errs, ValidationError{Field: prefix + ".license", Message: "must be an SPDX-like identifier without whitespace"})
		}
	}
	return errs
}

func validateURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("must be an http(s) URL")
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}
