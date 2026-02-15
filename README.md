# go-crawler

A concurrent web crawler and search backend written in Go. Designed for learning clean architecture and concurrency patterns—worker pools, channels, context cancellation, and mutex-protected stores.

## Features

- **Worker pool** — Configurable number of goroutines to crawl pages in parallel
- **URL deduplication** — Visits each URL at most once per crawl job
- **Depth limiting** — Respects `MaxDepth` to bound crawl depth from the start URL
- **Page limits** — Stops when `MaxPages` is reached
- **Same-domain only** — Optional restriction to links within the start URL’s host
- **In-memory storage** — `JobStore` and `PageStore` with mutex-protected access
- **Job lifecycle** — Status flow: `PENDING` → `RUNNING` → `COMPLETED` / `CANCELLED` / `FAILED`

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   HTTP      │────▶│   Service    │────▶│   Engine    │
│  (planned)  │     │  (lifecycle) │     │ (worker pool)│
└─────────────┘     └──────┬───────┘     └──────┬──────┘
                          │                     │
                          ▼                     ▼
                   ┌──────────────┐      ┌──────────────┐
                   │ JobStore     │      │ PageStore    │
                   │ PageStore    │      │ (via service)│
                   └──────────────┘      └──────────────┘
```

**Layers**

- **model** — Domain types: `CrawlJob`, `CrawlInput`, `URLTask`, `Page`
- **store** — Concurrency-safe in-memory stores for jobs and pages
- **service** — Lifecycle orchestration; owns status transitions
- **crawl** — Engine with worker pool, URL queue, HTML fetch, link extraction
- **cmd/server** — Entrypoint (HTTP API layer planned)

The service layer owns job lifecycle and status updates. The engine does not update job status directly; it uses interfaces (`PagesCrawledLimiter`, `PageWriter`) provided by the service layer.

## Project Structure

```
go-crawler/
├── cmd/
│   └── server/          # Main entrypoint
├── internal/
│   ├── model/           # Domain types
│   ├── store/           # JobStore, PageStore
│   ├── service/         # CrawlService, adapters
│   └── crawl/           # Engine, parser, deduplication
├── docs/
│   └── TEACHING-PLAN.md
└── go.mod
```

## Quick Start

**Prerequisites:** Go 1.25+

```bash
# Clone and enter the project
cd go-crawler

# Install dependencies
go mod download

# Run the server (submits a demo crawl to golang.org)
go run ./cmd/server
```

The demo submits a crawl with `MaxDepth=1`, `MaxPages=5`, `SameDomainOnly=true`, waits 15 seconds, then prints job status and crawled pages.

## Crawl Input

| Field           | Description                                  |
|----------------|----------------------------------------------|
| `StartURL`     | Seed URL for the crawl                       |
| `MaxDepth`     | Maximum depth from start (0 = start only)    |
| `MaxPages`     | Maximum number of pages to crawl             |
| `SameDomainOnly` | Restrict links to the start URL’s host    |
| `RequestDelayMs` | Delay between requests (0 = none)         |

## Dependencies

- [github.com/google/uuid](https://github.com/google/uuid) — Job and page IDs
- [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html) — HTML parsing and link extraction

## License

See repository for license details.
