-- name: UpsertPage :one
INSERT INTO pages (job_id, url, title, html, text_content)
VALUES (sqlc.arg(job_id), sqlc.arg(url), sqlc.arg(title), sqlc.arg(html), sqlc.arg(text_content))
ON CONFLICT (url) DO UPDATE SET
job_id = EXCLUDED.job_id,
title = EXCLUDED.title,
html = EXCLUDED.html,
text_content = EXCLUDED.text_content,
fetched_at = NOW()
RETURNING *;

-- name: GetPagesByJobID :many
SELECT * FROM pages WHERE job_id = sqlc.arg(job_id);
