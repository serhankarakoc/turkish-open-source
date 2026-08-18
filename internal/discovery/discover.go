package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
)

type Logger interface {
	Printf(format string, v ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

type Report struct {
	Queries             int
	InvalidQueries      int
	RawResults          int
	UniqueRepositories  int
	UniqueUsers         int
	UserReposEnumerated int
}

func Discover(ctx context.Context, client *gh.Client, cfg *config.Config, log Logger) (*Set, Report, error) {
	if log == nil {
		log = nopLogger{}
	}
	queries := BuildQueries(cfg)
	set := NewSet()
	rep := Report{Queries: len(queries)}

	maxPages := cfg.Search.MaxPagesPerQuery
	perPage := cfg.Search.ResultsPerPage

	for i, q := range queries {
		if err := ctx.Err(); err != nil {
			return set, rep, err
		}
		log.Printf("discovery [%d/%d] %s %s", i+1, len(queries), q.Kind, q.Q)
		switch q.Kind {
		case KindUserLocation, KindOwnerPhrase:
			users, err := client.SearchUsers(ctx, q.Q, maxPages, perPage)
			if err != nil {
				if errors.Is(err, gh.ErrInvalidSearchQuery) {
					rep.InvalidQueries++
				}
				log.Printf("search users %q: %v", q.Q, err)
				continue
			}
			rep.RawResults += len(users)
			for _, u := range users {
				set.AddUser(u)
			}
		default:
			repos, err := client.SearchRepositories(ctx, q.Q, maxPages, perPage)
			if err != nil {
				if errors.Is(err, gh.ErrInvalidSearchQuery) {
					rep.InvalidQueries++
				}
				log.Printf("search repos %q: %v", q.Q, err)
				continue
			}
			rep.RawResults += len(repos)
			for _, r := range repos {
				set.AddRepository(r, q.Source)
			}
		}
	}

	users := set.Users()
	maxUsers := cfg.Search.MaxUsersPerScan
	maxOrgs := cfg.Search.MaxOrgsPerScan
	if maxUsers <= 0 {
		maxUsers = 250
	}
	if maxOrgs <= 0 {
		maxOrgs = 150
	}
	trimmed := make([]gh.User, 0, len(users))
	userCount := 0
	orgCount := 0
	for _, u := range users {
		if strings.EqualFold(u.Type, "Organization") {
			if orgCount >= maxOrgs {
				continue
			}
			orgCount++
		} else {
			if userCount >= maxUsers {
				continue
			}
			userCount++
		}
		trimmed = append(trimmed, u)
	}
	users = trimmed

	for i, u := range users {
		if err := ctx.Err(); err != nil {
			return set, rep, err
		}
		log.Printf("user repos [%d/%d] %s", i+1, len(users), u.Login)
		repos, err := client.ListOwnerRepos(ctx, u.Login, u.Type, cfg.Search.MaxReposPerUser)
		if err != nil {
			log.Printf("list repos %s: %v", u.Login, err)
			continue
		}
		rep.UserReposEnumerated += len(repos)
		source := fmt.Sprintf("user:%s", u.Login)
		for _, r := range repos {
			set.AddRepository(r, source)
		}
	}

	rep.UniqueRepositories = set.Len()
	rep.UniqueUsers = set.UserCount()
	return set, rep, nil
}
