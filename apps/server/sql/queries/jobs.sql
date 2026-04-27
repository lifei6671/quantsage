-- name: ListJobRuns :many
SELECT id, job_name, biz_date, status, started_at, finished_at, error_code, error_message, retry_count, progress_current, progress_total, meta, created_at
FROM job_run_log
WHERE ($1::text = '' OR job_name = $1)
  AND ($2::date IS NULL OR biz_date = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetJobRun :one
SELECT id, job_name, biz_date, status, started_at, finished_at, error_code, error_message, retry_count, progress_current, progress_total, meta, created_at
FROM job_run_log
WHERE id = $1;
