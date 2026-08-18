package discovery

import (
	"testing"

	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
)

func TestCandidateDedupByID(t *testing.T) {
	set := NewSet()
	repo := gh.Repository{ID: 42, FullName: "acme/one", Name: "one"}
	if !set.AddRepository(repo, "topic:turkey") {
		t.Fatal("first insert should be new")
	}
	if set.AddRepository(repo, "location:Istanbul") {
		t.Fatal("same id must not insert twice")
	}
	if set.Len() != 1 {
		t.Fatalf("len=%d", set.Len())
	}
	got, ok := set.Get(42)
	if !ok {
		t.Fatal("missing candidate")
	}
	if len(got.Sources) != 2 {
		t.Fatalf("sources=%v", got.Sources)
	}
}

func TestCandidateDedupByFullName(t *testing.T) {
	set := NewSet()
	set.AddRepository(gh.Repository{ID: 1, FullName: "Acme/Lib"}, "a")
	if set.AddRepository(gh.Repository{ID: 2, FullName: "acme/lib"}, "b") {
		t.Fatal("same owner/name with a different id must not be stored twice")
	}
	if set.Len() != 1 {
		t.Fatalf("len=%d", set.Len())
	}
}

func TestUserDedup(t *testing.T) {
	set := NewSet()
	if !set.AddUser(gh.User{Login: "ada"}) {
		t.Fatal("expected first user")
	}
	if set.AddUser(gh.User{Login: "Ada"}) {
		t.Fatal("case-insensitive user dedup failed")
	}
	if set.UserCount() != 1 {
		t.Fatalf("users=%d", set.UserCount())
	}
}
