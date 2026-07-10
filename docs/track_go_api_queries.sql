-- Track Go API usage from auth_logs.dmama_use.
-- Run these queries on the PostgreSQL database configured by GISDATA_URL.

-- Latest requests.
SELECT started_at, api_key, method, path, status, duration_ms, size_value, size_unit, request_id
FROM auth_logs.dmama_use
ORDER BY started_at DESC
LIMIT 100;

-- Endpoint summary for the last 24 hours.
SELECT path, method, status, count(*) AS calls, avg(duration_ms)::int AS avg_ms
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY path, method, status
ORDER BY calls DESC;

-- Error/status rate for the last 24 hours.
SELECT status, count(*) AS calls
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY status
ORDER BY status;

-- Top API keys for the last 24 hours.
SELECT api_key, count(*) AS calls, sum(size_bytes) AS total_bytes, avg(duration_ms)::int AS avg_ms
FROM auth_logs.dmama_use
WHERE started_at >= now() - interval '1 day'
GROUP BY api_key
ORDER BY calls DESC;