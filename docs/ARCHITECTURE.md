# Architecture

V1 is a GitHub repository plus a Go scanner. There is no application database, web server, or frontend.

```text
GitHub API
    ↓
Discovery
    ↓
Deduplication
    ↓
Validation
    ↓
Scoring
    ↓
Categorization
    ↓
Dataset
    ↓
README generator
```

## Canonical data

| Path | Role |
|---|---|
| GitHub REST API | Source of repository metadata |
| `config/*.yml` | Queries, thresholds, categories, curated framework seeds |
| `data/projects.json` | Canonical catalog |
| `data/frameworks.json` | Featured frameworks scanned from `config/frameworks.yml` |
| `data/categories.json` | Generated category statistics |
| `data/contributors.json` | Reserved for later contributor tracking |
| `data/countries.json` | Country names and domains used by scoring |
| `README.md` | Generated view of the dataset |

The dataset is intentionally JSON so a later website, REST API, GraphQL API, search index, or dashboard can consume it without a database.

## Pipeline

1. **Discovery** builds search queries from `config/cities.yml` and `config/keywords.yml` (user location, owner/org phrases, topics, language+location, keywords, category-intent families). It does not rely on a single query.
2. **Deduplication** keys candidates by GitHub repository ID and keeps `full_name` as a secondary identifier.
3. **Validation** checks that the repository exists, is public, applies fork/archived policy, and records license status.
4. **Scoring** computes `turkey_score`, `quality_score`, and `activity_score`. One weak keyword cannot pass the Turkey threshold, and Turkey-related data repositories can receive negative evidence.
5. **Categorization** maps topics (and language fallbacks) to `config/categories.yml`, keeps a primary `category`, and may also persist `categories` for secondary matches.
6. **Dataset merge** updates GitHub-generated fields, preserves community-maintained fields (`is_verified` + `verification: community`, manual category), and never drops community-verified projects silently.
7. **Framework seed scan** reads `config/frameworks.yml` (GitHub URL, category, initial status only), fetches live repository metadata from the GitHub API, builds `country_evidence`, and writes `data/frameworks.json`.
8. **README generator** rewrites the block between `<!-- GENERATED:START -->` and `<!-- GENERATED:END -->`.

## GitHub client

`internal/github` is a reusable REST client:

- Configurable base URL and API version
- `GITHUB_TOKEN` via environment (never hardcoded)
- Timeouts and context cancellation
- Exponential backoff and `Retry-After`
- `X-RateLimit-*` awareness
- Bounded worker pool for enrichment (not unbounded fan-out)
- In-memory ETag cache for conditional `GET`s
- Sequential/low-concurrency search because search rate limits are stricter than the core API
- Owner / organization profile caching so expanded traversal does not re-fetch the same profile repeatedly

## Idempotency

Re-running the scanner against the same GitHub data and config yields a deterministic `projects.json` ordering:

1. category ascending
2. stars descending
3. name ascending

`generated_at` / `last_scanned_at` reflect the scan clock.

## Extending later

Keep V1 simple. When the JSON dataset is no longer enough, a future app can read `data/projects.json` (or generate it in CI) without changing the scoring rules. Do not introduce PostgreSQL, Redis, Docker, or a web UI in this repository until that split is justified.
