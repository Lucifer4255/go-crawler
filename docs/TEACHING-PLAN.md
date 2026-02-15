# go-crawler: Teaching Session Plan

A step-by-step curriculum to build the crawler while understanding concurrency patterns and clean architecture. Follow the project rules in `.cursor/rules/go-crawler-learning.mdc`.

---

## Architecture Reminder

```
User → HTTP (future) → Service → Engine
                           ↓
                    JobStore, PageStore
```

**Rules:** Service owns lifecycle; Engine does crawl work; interfaces in service layer; pointer receivers for mutex structs.

---

## Lesson 1: Active Task Counter & Termination Logic ✅

**Goal:** Fix the engine so workers know when there's no more work and exit cleanly.

**Concepts:**
- `activeCount`: tasks in queue + tasks being processed
- When does work "end"? When `activeCount == 0`
- Who closes the channel? The worker that decrements to 0 (only one can)
- `sync/atomic` for thread-safe counter

**Outcome:** Engine terminates correctly when queue drains.

---

## Lesson 2: HTTP Fetch (Single Page)

**Goal:** Fetch a page by URL.

**Concepts:**
- `net/http` Client with timeout
- Respect `RequestDelayMs` (e.g. `time.Sleep`)
- Handle errors (4xx, 5xx, timeout) gracefully

**Outcome:** Worker can fetch HTML for a URL.

---

## Lesson 3: Link Extraction

**Goal:** Parse HTML and extract links from `<a href="...">`.

**Concepts:**
- Use `goquery` or `golang.org/x/net/html` for parsing
- Normalize URLs (resolve relative → absolute)
- Filter: same-domain-only if `SameDomainOnly` is true

**Outcome:** Given HTML + base URL, return list of absolute URLs.

---

## Lesson 4: Enforce Limits (MaxDepth, MaxPages)

**Goal:** Stop crawling when limits are hit.

**Concepts:**
- Check `task.Depth <= job.Input.MaxDepth` before processing
- Check `job.PagesCrawled < job.Input.MaxPages` (but Engine must NOT update job directly—how?)
- Options: Engine receives a callback, or Engine reads current value from a shared source the Service updates

**Design question:** How does Engine learn "pages crawled" without updating job status? (Hint: Service could pass an interface for "increment" that Engine calls; Service updates job.)

**Outcome:** Crawl respects depth and page limits.

---

## Lesson 5: PageStore Integration

**Goal:** Save each crawled page to PageStore.

**Concepts:**
- Engine needs PageStore (inject via constructor or Start params)
- Extract title and content (strip HTML tags for content)
- Create `model.Page` and call `CreatePage`

**Outcome:** Crawled pages persist in PageStore.

---

## Lesson 6: Service ↔ Engine Wiring

**Goal:** When user submits a crawl, the engine actually runs.

**Concepts:**
- Service receives Engine (or Engine interface)
- `Submit` creates job → stores it → starts Engine in a goroutine
- Service updates job: PENDING → RUNNING before Start, COMPLETED/FAILED after
- Use `context.Context` for cancellation

**Outcome:** Full crawl flow works end-to-end (no HTTP yet).

---

## Lesson 7: HTTP API Layer

**Goal:** Expose REST endpoints.

**Endpoints:**
- `POST /crawl` — submit crawl (body: CrawlInput)
- `GET /crawl/:id` — job status
- `GET /crawl/:id/pages` — list pages for job

**Concepts:**
- Chi, Echo, or net/http ServeMux
- Wire Service into handlers
- Replace `main.go` placeholder with server

**Outcome:** Can submit and query via HTTP.

---

## Lesson 8 (Optional): Search Index & Query

**Goal:** Build index from Page content, support term search.

**Concepts:**
- IndexStore: term → []pageID
- Tokenize Page.Content (split, lowercase, dedupe)
- Search: lookup term → fetch pages

**Outcome:** "Search a term" returns matching pages.

---

## Order of Study

| # | Lesson                    | Depends On | Status   |
|---|---------------------------|------------|----------|
| 1 | Active task counter       | —          | Ready    |
| 2 | HTTP fetch                | 1          | Pending  |
| 3 | Link extraction           | 2          | Pending  |
| 4 | MaxDepth / MaxPages       | 1, 3       | Pending  |
| 5 | PageStore integration     | 2, 3       | Pending  |
| 6 | Service ↔ Engine wiring   | 1–5        | Pending  |
| 7 | HTTP API                  | 6          | Pending  |
| 8 | Search index (optional)   | 5, 7       | Pending  |

---

## How to Use This Plan

1. Complete each lesson before moving on.
2. Run `go build ./...` and tests after each change.
3. Ask guiding questions if stuck; refer to project rules for design constraints.
