package framework

import "fmt"

func ValidateDataset(ds Dataset) []error {
	var errs []error
	if ds.Version != 0 && ds.Version != DatasetVersion {
		errs = append(errs, fmt.Errorf("frameworks version %d unsupported", ds.Version))
	}
	seen := map[string]struct{}{}
	for i, f := range ds.Frameworks {
		if f.GitHub == "" {
			errs = append(errs, fmt.Errorf("frameworks[%d]: missing github", i))
			continue
		}
		key := CanonicalKey(f.GitHub)
		if _, ok := seen[key]; ok {
			errs = append(errs, fmt.Errorf("duplicate github %s", f.GitHub))
		}
		seen[key] = struct{}{}
		if f.Status != "" && !validOutputStatus(f.Status) {
			errs = append(errs, fmt.Errorf("%s: invalid status %q", f.GitHub, f.Status))
		}
		if f.Stars < 0 {
			errs = append(errs, fmt.Errorf("%s: stars must be >= 0", f.GitHub))
		}
		if f.CountryScore < 0 || f.CountryScore > 100 {
			errs = append(errs, fmt.Errorf("%s: country_score must be 0..100", f.GitHub))
		}
	}
	return errs
}

func validOutputStatus(status string) bool {
	switch status {
	case StatusVerified, StatusPendingVerification, StatusHistorical, StatusNotFound, StatusExcluded:
		return true
	default:
		return false
	}
}
