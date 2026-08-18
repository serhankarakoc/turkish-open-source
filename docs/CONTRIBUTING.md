# Contribution guide

## How the catalog is updated

1. The scanner queries the GitHub REST API.
2. Candidates are deduplicated by GitHub repository ID.
3. License, activity, and Turkey signals are validated.
4. Qualifying projects are merged into `data/projects.json`.
5. `README.md` is generated from that dataset.

You usually do **not** need to send a pull request that only lists a repository in Markdown.

## Add a project

Preferred path:

1. Open an issue with the **Add Project** template.
2. Provide the GitHub URL, license, category, and **independent** reasons the project is Turkish.
3. Maintainers may mark the entry `verification: community` after review.

Optional path for maintainers:

1. Add or update the project in `data/projects.json`.
2. Keep GitHub-generated fields consistent with the API.
3. Set community fields explicitly:

```json
{
  "is_verified": true,
  "verification": "community"
}
```

4. Run `go run ./cmd/scanner --generate` to refresh the README.
5. Do not duplicate IDs or `owner/name` pairs.

## Report a problem

Use the **Report Project** template for incorrect classification, inactivity, duplicates, wrong category, misleading Turkey score, or licensing issues.

## Local development

```bash
go test ./...
go vet ./...
go run ./cmd/scanner --validate
go run ./cmd/scanner --dry-run --verbose
go run ./cmd/scanner --generate
```

`GITHUB_TOKEN` is required for discovery. Do not commit tokens.

## What will be rejected

- Spam, advertising, or empty placeholder repositories
- Proprietary / commercial-only code presented as open source
- Repositories without a recognized license while `projects.require_license` is `true`
- Duplicates (same GitHub ID or the same `owner/name`)
- Projects whose only Turkey signal is a single weak keyword
- Archived projects in active rankings when `include_archived` is `false`
- Secrets, credentials, or generated-section hand edits

Maintainers may reject a project even if the automated score is high.

## Community verification

Automated `verification: "automated"` means high-confidence **heuristics**, not official certification.

If a maintainer sets `verification: "community"` and `is_verified: true`, the scanner must preserve those fields. A later automated score change must not silently un-verify the project.
