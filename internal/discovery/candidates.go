package discovery

import (
	"strings"
	"sync"

	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
)

type Candidate struct {
	Repository gh.Repository
	Sources    []string
}

type Set struct {
	mu         sync.Mutex
	byID       map[int64]*Candidate
	byFullName map[string]int64
	users      map[string]gh.User
	userOrder  []string
}

func NewSet() *Set {
	return &Set{
		byID:       map[int64]*Candidate{},
		byFullName: map[string]int64{},
		users:      map[string]gh.User{},
	}
}

func (s *Set) AddRepository(repo gh.Repository, source string) bool {
	if repo.ID == 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.byID[repo.ID]; ok {
		existing.Sources = appendUnique(existing.Sources, source)
		if len(existing.Repository.Topics) == 0 && len(repo.Topics) > 0 {
			existing.Repository.Topics = repo.Topics
		}
		if existing.Repository.Description == "" && repo.Description != "" {
			existing.Repository.Description = repo.Description
		}
		return false
	}
	key := strings.ToLower(repo.FullName)
	if key != "" {
		if id, ok := s.byFullName[key]; ok && id != repo.ID {
			return false
		}
		s.byFullName[key] = repo.ID
	}
	s.byID[repo.ID] = &Candidate{
		Repository: repo,
		Sources:    appendUnique(nil, source),
	}
	return true
}

func (s *Set) AddUser(user gh.User) bool {
	if user.Login == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(user.Login)
	if _, ok := s.users[key]; ok {
		return false
	}
	s.users[key] = user
	s.userOrder = append(s.userOrder, user.Login)
	return true
}

func (s *Set) Users() []gh.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]gh.User, 0, len(s.userOrder))
	for _, login := range s.userOrder {
		if u, ok := s.users[strings.ToLower(login)]; ok {
			out = append(out, u)
		}
	}
	return out
}

func (s *Set) Candidates() []*Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Candidate, 0, len(s.byID))
	for _, c := range s.byID {
		out = append(out, c)
	}
	return out
}

func (s *Set) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byID)
}

func (s *Set) UserCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.users)
}

func (s *Set) Get(id int64) (*Candidate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[id]
	return c, ok
}

func appendUnique(in []string, v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return in
	}
	for _, existing := range in {
		if existing == v {
			return in
		}
	}
	return append(in, v)
}
