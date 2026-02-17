-- name: GetJob :one
SELECT * FROM jobs WHERE id = sqlc.arg(id);

-- name: CreateJob :one
INSERT INTO jobs (id, input, status, error, pages_crawled, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(input), sqlc.arg(status), sqlc.arg(error), sqlc.arg(pages_crawled), sqlc.arg(created_at), sqlc.arg(updated_at)) RETURNING *;

-- name: UpdateJobStatus :one
UPDATE jobs SET status = sqlc.arg(status), error = sqlc.arg(error), updated_at = NOW() WHERE id = sqlc.arg(id) RETURNING *;

-- name: TryIncrementPagesCrawled :one
UPDATE jobs SET pages_crawled = pages_crawled + 1, updated_at = NOW()
WHERE id = sqlc.arg(id) AND pages_crawled < sqlc.arg(max_pages) RETURNING *;

-- name: GetAllJobs :many
SELECT * FROM jobs;