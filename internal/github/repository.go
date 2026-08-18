package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Repository struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	FullName         string    `json:"full_name"`
	URL              string    `json:"url"`
	HTMLURL          string    `json:"html_url"`
	Description      string    `json:"description"`
	Language         string    `json:"language"`
	Homepage         string    `json:"homepage"`
	StargazersCount  int       `json:"stargazers_count"`
	ForksCount       int       `json:"forks_count"`
	OpenIssuesCount  int       `json:"open_issues_count"`
	WatchersCount    int       `json:"watchers_count"`
	SubscribersCount int       `json:"subscribers_count"`
	DefaultBranch    string    `json:"default_branch"`
	Topics           []string  `json:"topics"`
	Fork             bool      `json:"fork"`
	MirrorURL        string    `json:"mirror_url"`
	Archived         bool      `json:"archived"`
	Disabled         bool      `json:"disabled"`
	Private          bool      `json:"private"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	PushedAt         time.Time `json:"pushed_at"`
	License          *License  `json:"license"`
	Owner            *User     `json:"owner"`
}

type License struct {
	Key    string `json:"key"`
	SPDXID string `json:"spdx_id"`
	Name   string `json:"name"`
}

type User struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Type     string `json:"type"`
	HTMLURL  string `json:"html_url"`
	Name     string `json:"name"`
	Company  string `json:"company"`
	Blog     string `json:"blog"`
	Location string `json:"location"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
	Twitter  string `json:"twitter_username"`
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	var r Repository
	path := fmt.Sprintf("/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo))
	if err := c.GetJSON(ctx, path, nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) GetUser(ctx context.Context, login string) (*User, error) {
	var u User
	path := "/users/" + url.PathEscape(login)
	if err := c.GetJSON(ctx, path, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *Client) GetOrganization(ctx context.Context, org string) (*User, error) {
	var u User
	path := "/orgs/" + url.PathEscape(org)
	if err := c.GetJSON(ctx, path, nil, &u); err != nil {
		return nil, err
	}
	if u.Type == "" {
		u.Type = "Organization"
	}
	return &u, nil
}

func (c *Client) GetLanguages(ctx context.Context, owner, repo string) (map[string]int64, error) {
	var langs map[string]int64
	path := fmt.Sprintf("/repos/%s/%s/languages", url.PathEscape(owner), url.PathEscape(repo))
	if err := c.GetJSON(ctx, path, nil, &langs); err != nil {
		if IsNotFound(err) {
			return map[string]int64{}, nil
		}
		return nil, err
	}
	if langs == nil {
		langs = map[string]int64{}
	}
	return langs, nil
}

func PrimaryLanguage(langs map[string]int64) string {
	best := ""
	var bytes int64
	for lang, n := range langs {
		if lang == "" || n <= bytes {
			continue
		}
		bytes = n
		best = lang
	}
	return best
}

func (c *Client) ListUserRepos(ctx context.Context, login string, maxRepos int) ([]Repository, error) {
	return c.ListOwnerRepos(ctx, login, "User", maxRepos)
}

func (c *Client) ListOwnerRepos(ctx context.Context, login, ownerType string, maxRepos int) ([]Repository, error) {
	if maxRepos <= 0 {
		maxRepos = 100
	}
	var out []Repository
	page := 1
	perPage := 100
	if maxRepos < perPage {
		perPage = maxRepos
	}
	for len(out) < maxRepos {
		q := url.Values{}
		q.Set("type", "owner")
		q.Set("sort", "updated")
		q.Set("direction", "desc")
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		var pageRepos []Repository
		path := "/users/" + url.PathEscape(login) + "/repos"
		if strings.EqualFold(ownerType, "Organization") {
			path = "/orgs/" + url.PathEscape(login) + "/repos"
		}
		if err := c.GetJSON(ctx, path, q, &pageRepos); err != nil {
			return out, err
		}
		if len(pageRepos) == 0 {
			break
		}
		out = append(out, pageRepos...)
		if len(pageRepos) < perPage {
			break
		}
		page++
	}
	if len(out) > maxRepos {
		out = out[:maxRepos]
	}
	return out, nil
}

func (c *Client) GetReadme(ctx context.Context, owner, repo string) (string, error) {
	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	path := fmt.Sprintf("/repos/%s/%s/readme", url.PathEscape(owner), url.PathEscape(repo))
	if err := c.GetJSON(ctx, path, nil, &payload); err != nil {
		if IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if !strings.EqualFold(payload.Encoding, "base64") {
		return payload.Content, nil
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, payload.Content)
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(cleaned)
		if err != nil {
			return "", err
		}
	}
	text := string(decoded)
	if len(text) > 32*1024 {
		text = text[:32*1024]
	}
	return text, nil
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	var rel Release
	path := fmt.Sprintf("/repos/%s/%s/releases/latest", url.PathEscape(owner), url.PathEscape(repo))
	if err := c.GetJSON(ctx, path, nil, &rel); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if rel.Draft {
		return nil, nil
	}
	return &rel, nil
}

func (c *Client) CountContributors(ctx context.Context, owner, repo string) (int, error) {
	logins, err := c.ListContributorLogins(ctx, owner, repo, 100)
	if err != nil {
		return 0, err
	}
	return len(logins), nil
}

func (c *Client) ListContributorLogins(ctx context.Context, owner, repo string, max int) ([]string, error) {
	if max <= 0 {
		max = 3
	}
	if max > 100 {
		max = 100
	}
	q := url.Values{}
	q.Set("per_page", strconv.Itoa(max))
	q.Set("anon", "false")
	path := fmt.Sprintf("/repos/%s/%s/contributors", url.PathEscape(owner), url.PathEscape(repo))
	var people []struct {
		Login string `json:"login"`
	}
	if err := c.GetJSON(ctx, path, q, &people); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(people))
	for _, p := range people {
		if strings.TrimSpace(p.Login) == "" {
			continue
		}
		out = append(out, p.Login)
	}
	return out, nil
}

func (r Repository) OwnerLogin() string {
	if r.Owner == nil {
		parts := strings.SplitN(r.FullName, "/", 2)
		if len(parts) > 0 {
			return parts[0]
		}
		return ""
	}
	return r.Owner.Login
}

func (r Repository) OwnerType() string {
	if r.Owner == nil {
		return ""
	}
	return r.Owner.Type
}

func (r Repository) LicenseID() string {
	if r.License == nil {
		return ""
	}
	if r.License.SPDXID != "" && !strings.EqualFold(r.License.SPDXID, "NOASSERTION") {
		return r.License.SPDXID
	}
	return r.License.Key
}

func decodeJSON(raw []byte, dest any) error {
	return json.Unmarshal(raw, dest)
}
