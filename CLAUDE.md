# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

PowerLive.io is a fixed-project download portal based on Yatko. It auto-detects
the visitor's platform/arch and redirects to the right release asset. The
frontend and backend can be deployed as separate Coolify services:

- `backend/` — Go + Gin API, deployed as a container-image service (`backend/Dockerfile`)
- `frontend/` — Next.js 16 (App Router, React 19), deployed as the default Node.js service

Root-level `vercel.json` remains available for Vercel deployments. In Coolify,
the frontend's Next rewrites proxy `/dl`, `/api`, `/code`, and `/health` to
`BACKEND_URL`.

## Commands

Run both services together for local dev:

```bash
./dev.sh
```

### Backend (`backend/`)

```bash
go run .                          # start the API on :8080
go test ./...                     # run all tests
go test ./cache/ -run TestName    # run a single test
go build ./...                    # compile check
```

Useful backend env vars: `GITHUB_TOKEN`, `OSS_ENDPOINT`, `OSS_BUCKET`,
`OSS_ENDPOINT`, `OSS_BUCKET`, `OSS_ACCESS_KEY_ID`, `OSS_ACCESS_KEY_SECRET`, `RELEASE_CDN_URL`,
`RELEASE_SYNC_INTERVAL`, `RELEASE_SYNC_LIMIT`, `ALLOWED_REPOS`,
`TRUSTED_PROXIES`, `REDIS_URL`, `CACHE_TTL_SECONDS`,
`CACHE_REFRESH_SECRET`, `RATE_LIMIT_RPM`, and `PORT`. See
`docs/backend-release-sync.md` for the OSS worker configuration.

### Frontend (`frontend/`)

```bash
bun install
bun run dev      # Next.js dev server on :3000
bun run build
bun run start
```

Requires `BACKEND_URL` (server-side only for release pages — every backend call under `frontend/app/p/` is server-side). The homepage autocomplete calls same-origin `/api/search`, which Vercel rewrites to the backend (and `next.config.ts` proxies to `BACKEND_URL` in local `next dev`). For local dev outside `./dev.sh`, set `BACKEND_URL` to `http://localhost:8080`.

There is no lint script and no test runner configured on the frontend.

## Backend architecture

Request flow for a download: `main.go` wires a single `github.Client` and `cache.Cache` into each handler; handlers stay thin and delegate to these packages.

- **`github/`** — GitHub REST API client (releases by tag/latest, README fetch, repo search). Returns `*github.APIError` with a `StatusCode` on failure; `handlers/errors.go` maps that to the outward-facing HTTP status (404/403/429/502). Search uses a separate request path that does not update the core rate-limit budget (Search API remaining is ~30/min and would falsely trip the core reserve of 200).
- **`cache/`** — Redis-backed cache wrapping every GitHub fetch through `cache.FetchCached`. Falls back to a graceful no-op when `UPSTASH_REDIS_URL` is unset, so it works without Redis in local dev. Design (read before changing cache behavior):
 - **Soft TTL** (`CACHE_TTL_SECONDS`, default 1 hour): serve cached value as-is, zero network calls. Search keys use a longer soft TTL (2h).
 - **L1 (in-process)**: hot keys are also kept in a small process-local LRU so repeated `/dl` hits skip Upstash.
 - **Revalidation**: after soft TTL, serve the stale value immediately and revalidate in the background with `If-None-Match`; a `304` does not count against GitHub's rate limit and extends freshness.
 - **Hard TTL** (24h): how long a stale entry (and its ETag) is kept so revalidation stays possible.
  - **Request coalescing**: concurrent misses for the same key are deduplicated via `singleflight`.
  - **Stale-while-error**: if GitHub is unreachable or rate-limited, serve the last known-good value when one exists.
  - **Rate-limit budget guard**: the GitHub client refuses new requests once `X-RateLimit-Remaining` drops below a reserve (200), so bursts of cold repos fail fast with 429 instead of starving already-cached ones. Tracks `X-RateLimit-Reset` and clears the observed budget once that window ends so the guard doesn't stick until process restart.
- **`picker/`** — pure platform/arch detection and asset-matching logic (User-Agent parsing → OS/arch → best-matching release asset). No I/O.
- **`releasesync/`** — long-running GitHub → OSS release mirror. Polls the allowlisted repositories, streams missing assets to OSS, and writes `_manifest.json` last.
- **`cdn/`** — rewrites an asset to `RELEASE_CDN_URL` only when the completion manifest is reachable, otherwise preserving the GitHub URL.
- **`handlers/`** — Gin route handlers, one file per route family (`redirect`, `link`, `page`, `releases`, `search`). Each handler fetches via `cache.FetchCached(ctx, cache, key, ghClientCall)`, then hands assets to `picker` if it needs to pick one.
- **`middleware/`** — per-IP HTTP rate limit (`RATE_LIMIT_RPM`, default 120/min); no-ops without `UPSTASH_REDIS_URL`. When Redis is configured but unreachable, falls back to a process-local fixed window instead of failing open. Separate from the GitHub cache/rate-limit budget above. `/api/search` also has a tighter prefixed budget (`SEARCH_RATE_LIMIT_RPM`, default 20).

No CORS policy: the deployment proxy presents frontend and backend under one
public origin. `main.go` only honors forwarded client IPs from
`TRUSTED_PROXIES`; leave it unset rather than trusting an unknown proxy range.

## Frontend architecture

App Router structure: `app/page.tsx` is the homepage/search; `app/p/[owner]/[repo]/` is the release landing page, with `[version]/page.tsx` for a pinned version. Route-scoped components (download button, version selector, checksum display, install-command extraction, share links, etc.) live alongside the pages they belong to rather than in a shared `components/` tree — `use-releases.ts` and `platform-utils.ts` in that directory are the shared hook/util for that route family.

The frontend calls the Go backend (`BACKEND_URL`/`NEXT_PUBLIC_BACKEND_URL`), not GitHub directly.

**Important:** `frontend/AGENTS.md` (pulled in via `frontend/CLAUDE.md`) warns that this Next.js version has breaking API/convention changes from training data — check `node_modules/next/dist/docs/` before writing Next.js-specific code in `frontend/`.

## graphify

This project has a knowledge graph at `graphify-out/` with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when `graphify-out/graph.json` exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than `GRAPH_REPORT.md` or raw grep output.
- If `graphify-out/wiki/index.md` exists, use it for broad navigation instead of raw source browsing.
- Read `graphify-out/GRAPH_REPORT.md` only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
