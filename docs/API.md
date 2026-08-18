# Scanner and GitHub API

## Authentication

The scanner uses the GitHub REST API.

| Variable | Required | Purpose |
|---|---|---|
| `GITHUB_TOKEN` | For discovery/update | Bearer token. In GitHub Actions this is `secrets.GITHUB_TOKEN`. |
| `GITHUB_API_URL` | No | Override API base URL (default `https://api.github.com` or `config/settings.yml`). |
| `GITHUB_API_VERSION` | No | Override `X-GitHub-Api-Version` (default `2026-03-10`). |

Do not hardcode personal access tokens. Do not commit `.env` files.

Request headers:

```text
Accept: application/vnd.github+json
Authorization: Bearer <token>
X-GitHub-Api-Version: 2026-03-10
User-Agent: turkish-open-source-scanner
```

The client never logs authorization headers or token values.

## Rate limits

Authenticated REST requests have a much higher quota than anonymous ones. Search endpoints have a **separate**, stricter limit.

The client:

1. Honors `Retry-After` when present.
2. If `X-RateLimit-Remaining` is `0`, waits until `X-RateLimit-Reset`.
3. Uses exponential backoff for secondary rate limits and `5xx`.
4. Stops after `scanner.max_retries`.
5. Does **not** retry typical `4xx` errors (`400`, `401`, `404`, `422`).
6. Treats some `403` responses as rate limits when GitHub says so; otherwise they fail.

Do not raise `scanner.max_workers` aggressively. GitHub secondary limits punish bursty concurrency. Default workers: 6.

## Configuration

All thresholds live under `config/`:

- `settings.yml` — workers, retries, timeouts, inclusion policy, API version
- `cities.yml` — location discovery strings
- `keywords.yml` — topics, keywords, search phrases, intent families, languages, stopwords
- `categories.yml` — category topics and labels
- `frameworks.yml` — curated framework seeds (`github`, `category`, `initial_status`)

Application code should not hardcode those values.

## Commands

```bash
go run ./cmd/scanner
go run ./cmd/scanner --dry-run
go run ./cmd/scanner --dry-run --verbose
go run ./cmd/scanner --discover
go run ./cmd/scanner --update
go run ./cmd/scanner --frameworks
go run ./cmd/scanner --frameworks --generate
go run ./cmd/scanner --generate
go run ./cmd/scanner --validate
```

| Flag | Effect |
|---|---|
| (default) | Discover catalog, scan framework seeds, merge, write datasets, generate README |
| `--dry-run` | Query GitHub and print a report; write nothing |
| `--discover` | Run discovery (full pipeline still scores/filters) |
| `--update` | Refresh existing dataset entries from GitHub without new search queries |
| `--frameworks` | Scan `config/frameworks.yml` seeds and refresh `data/frameworks.json` |
| `--generate` | Rebuild README (and category stats) from `data/projects.json` / `data/frameworks.json` |
| `--validate` | Schema/integrity checks; no GitHub calls |
| `--verbose` | Log request paths/status and stage progress (never tokens) |

`--generate` and `--validate` do not need `GITHUB_TOKEN`. `--frameworks` works with the public API if the token is unset, but authenticated requests have a much higher quota.

## Featured frameworks

`config/frameworks.yml` stores only `github`, `category`, `initial_status`, and an optional display `name`. The scanner never reads stars, language, license, or website from the seed file.

Each scan calls `GET /repos/{owner}/{repo}` (and owner/org, languages, README as needed) and writes live fields to `data/frameworks.json`. A 404 is recorded as `status: repository_not_found` and does not stop the scan.

GitHub Actions workflow `.github/workflows/update-frameworks.yml` runs this daily at 03:00 UTC.

## Discovery notes

The scanner does more than search for the words `turkey` / `turkiye`.

It combines:

- repository search
- owner / organization search
- topic search
- category-intent search families (`framework`, `library`, `devtools`, `api`, `ai`, `security`, etc.)
- owner/org expansion when a profile has Turkey-linked signals

This helps the catalog find Turkish-origin frameworks and developer tools that do not explicitly brand themselves with `Turkey` in the repository name.

## Local dry-run

```bash
export GITHUB_TOKEN=YOUR_GITHUB_TOKEN
go run ./cmd/scanner --dry-run --verbose
```

On Windows PowerShell:

```powershell
$env:GITHUB_TOKEN = "YOUR_GITHUB_TOKEN"
go run ./cmd/scanner --dry-run --verbose
```

If the token is missing, discovery exits with an error instead of inventing projects.

## Pagination

Search pagination is real, but bounded:

```yaml
search:
  max_pages_per_query: 5
  results_per_page: 100
```

The client follows `Link: rel="next"` until that cap. It does not blindly crawl hundreds of pages.

## Efficiency

The scanner avoids `N repositories × 5 requests` by:

- Deduplicating candidates in memory before enrichment
- Looking up existing `data/projects.json` rows
- Caching owner profiles
- Using conditional `GET` with ETags when GitHub returns `304`
- Fetching README/release only after cheap filters (fork, archive, license, preliminary score)
- Using a 4–8 worker pool for core API enrichment

## Dataset contract

`data/projects.json`:

```json
{
  "version": 1,
  "generated_at": "2026-08-17T02:00:00Z",
  "projects": []
}
```

Consumers (future website or API) should treat `id` as stable and `full_name` as a secondary key that can change on rename.

Projects may include additional backward-compatible metadata such as:

- `quality_score`
- `status`
- `evidence`
- `source`
- `categories`
