# Inclusion criteria

A catalog entry must be a **public, licensed, GitHub-hosted software project** with **independent evidence** that it is developed in or strongly associated with Turkey.

Detection is **heuristic**. Scores are not a legal nationality check, a government registry, or an official certification.

## Open source

A public GitHub repository is **not** automatically open source.

The scanner reads the GitHub license metadata (`license.spdx_id`). Recognized licenses include:

- MIT
- Apache-2.0
- GPL-2.0 / GPL-3.0
- LGPL-2.1 / LGPL-3.0
- BSD-2-Clause / BSD-3-Clause
- MPL-2.0
- ISC
- EPL-2.0
- AGPL-3.0

If GitHub reports no license, `license_status` is `unknown`. Those projects are **not** verified open source.

When `config/settings.yml` has `projects.require_license: true`, unlicensed repositories are excluded from the catalog (community-verified existing entries are still preserved).

When `config/settings.yml` has `projects.minimum_stars` set, the scanner excludes repositories with fewer stars than that threshold (community-verified existing entries are still preserved).

## Turkey association

The scanner never classifies a repository as Turkish from a **single weak signal** (for example one occurrence of the word "turkey" in a description).

Independent signal groups include:

| Group | Examples | Typical points |
|---|---|---|
| Owner location | User/org `location` contains Turkey or a configured city | +30 |
| README language | Substantial Turkish text in the README | +15 |
| Topics | `turkey`, `turkiye`, `turkish`, `made-in-turkey` | +10 |
| Website | Homepage/blog on a `.tr` domain | +5 |
| Maintainers | Multiple collaborators with Turkey location signals | +10 |
| Profile text | Company/bio mentions Turkey (supporting evidence only) | +5–10 |

Scores are capped at **100**. Stars and programming language are **not** Turkey signals.

The scanner also computes a separate `quality_score` so a repository can be Turkish-origin without automatically ranking highly. This reduces false positives from Turkey-related data/content repositories.

### Score bands

| Score | Band | Meaning |
|---:|---|---|
| 90–100 | `verified` | High-confidence **automated** verification |
| 75–89 | `strong` | Multiple independent signals |
| 50–74 | `possible` | Insufficient for default inclusion |
| 0–49 | `reject` | Missing or single weak evidence |

Default inclusion uses `projects.minimum_turkey_score: 75`.

By default `projects.minimum_stars: 10` acts as a quality guardrail, but the scanner may still keep low-star repositories when they have strong Turkey evidence plus strong framework/library/tool quality signals. This is important for newer projects.

`verification: "automated"` is not a claim of official status. Community review may set `verification: "community"`.

## Activity

Activity is scored separately from Turkey association. Typical signals:

- Recent `pushed_at` / updates
- A recent non-draft release
- Multiple contributors (when enrichment is enabled)
- Recent issue activity

Archived repositories are marked `is_archived: true` and `is_active: false`. They do not appear in active rankings. Forks are excluded unless `include_forks: true`.

## Unique identity

- Primary key: GitHub repository `id`
- Secondary key: `owner/name` (`full_name`)

The same repository discovered by many searches is stored once.

## Featured frameworks

`config/frameworks.yml` is a curated seed list. Manual fields are only the GitHub URL, category, and initial status. Stars, language, license, website, and Turkey evidence are always taken from the GitHub API.

Framework `status` values:

- `verified` — framework confirmed and Turkey evidence is strong
- `pending_verification` — looks like a framework, Turkey evidence is weak
- `historical` — real framework that is no longer the active product line
- `repository_not_found` — GitHub returned 404 for the seed URL
- `excluded` — does not meet framework criteria

## Preservation

Manually community-verified projects are never silently deleted or un-verified because an automated score changed.
