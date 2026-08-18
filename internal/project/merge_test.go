package project

import "testing"

func TestMergePreservesCommunityVerification(t *testing.T) {
	existing := Project{
		ID: 1, Name: "old", Stars: 10, IsVerified: true, Verification: "community",
		Category: "web", Categories: []string{"web"}, ManualCategory: true, FirstDiscoveredAt: "2024-01-01T00:00:00Z", Source: "manual",
	}
	incoming := Project{
		ID: 1, Name: "new", Stars: 12, IsVerified: false, Verification: "automated",
		Category: "other", TurkeyScore: 40, LastScannedAt: "2026-08-17T00:00:00Z", Categories: []string{"other"},
	}
	got := Merge(existing, incoming)
	if !got.IsVerified || got.Verification != "community" {
		t.Fatalf("community verification lost: %+v", got)
	}
	if got.Category != "web" || !got.ManualCategory {
		t.Fatalf("manual category lost: %+v", got)
	}
	if got.FirstDiscoveredAt != "2024-01-01T00:00:00Z" {
		t.Fatalf("first seen lost: %s", got.FirstDiscoveredAt)
	}
	if got.Stars != 12 || got.StarDelta != 2 {
		t.Fatalf("github fields not updated: stars=%d delta=%d", got.Stars, got.StarDelta)
	}
	if got.Source != "manual" || len(got.Categories) == 0 || got.Categories[0] != "other" {
		t.Fatalf("expected manual source preservation and incoming categories, got %+v", got)
	}
}

func TestMergeAllKeepsMissingCommunityProjects(t *testing.T) {
	existing := []Project{
		{ID: 1, Name: "Keep", FullName: "acme/keep", IsVerified: true, Verification: "community", Category: "other"},
		{ID: 2, Name: "Drop", FullName: "acme/drop", Category: "other"},
	}
	incoming := []Project{
		{ID: 3, Name: "New", FullName: "acme/new", Category: "ai", Stars: 5},
	}
	got := MergeAll(existing, incoming, true)
	ids := map[int64]bool{}
	for _, p := range got {
		ids[p.ID] = true
	}
	if !ids[1] || !ids[3] {
		t.Fatalf("expected community + new, got %+v", got)
	}
	if ids[2] {
		t.Fatal("unverified missing project should be removable")
	}
}

func TestSortProjectsDeterministic(t *testing.T) {
	projects := []Project{
		{Name: "b", Category: "web", Stars: 10, FullName: "z/b"},
		{Name: "a", Category: "web", Stars: 10, FullName: "a/a"},
		{Name: "c", Category: "ai", Stars: 50, FullName: "c/c"},
	}
	SortProjects(projects)
	if projects[0].Category != "ai" || projects[1].Name != "a" || projects[2].Name != "b" {
		t.Fatalf("unexpected order: %+v", projects)
	}
}

func TestValidateDatasetDuplicates(t *testing.T) {
	ds := Dataset{Version: 1, Projects: []Project{
		{ID: 1, Name: "a", FullName: "o/a", Owner: "o", HTMLURL: "https://github.com/o/a", URL: "https://api.github.com/repos/o/a", Category: "other", LicenseStatus: "valid"},
		{ID: 1, Name: "b", FullName: "o/b", Owner: "o", HTMLURL: "https://github.com/o/b", URL: "https://api.github.com/repos/o/b", Category: "other", LicenseStatus: "valid"},
	}}
	errs := ValidateDataset(ds, map[string]struct{}{"other": {}})
	if len(errs) == 0 {
		t.Fatal("expected duplicate id error")
	}
}
