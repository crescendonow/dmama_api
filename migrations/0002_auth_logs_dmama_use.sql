-- 0002_auth_logs_dmama_use.sql
-- Provision API usage logging storage. Run once on the PostgreSQL database configured by
-- GISDATA_URL with a role that can create schemas/tables and grant privileges.
--
-- Idempotent: safe to re-run.

CREATE SCHEMA IF NOT EXISTS auth_logs;

CREATE TABLE IF NOT EXISTS auth_logs.dmama_use (
    id          bigserial PRIMARY KEY,
    api_key     text,
    method      text,
    path        text,
    status      integer,
    size_bytes  bigint,
    size_value  double precision,
    size_unit   text,
    duration_ms bigint,
    started_at  timestamptz,
    ended_at    timestamptz,
    request_id  text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS dmama_use_api_key_idx ON auth_logs.dmama_use (api_key);
CREATE INDEX IF NOT EXISTS dmama_use_started_at_idx ON auth_logs.dmama_use (started_at);

GRANT USAGE ON SCHEMA auth_logs TO dmama;
GRANT SELECT, INSERT ON TABLE auth_logs.dmama_use TO dmama;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA auth_logs TO dmama;

ALTER DEFAULT PRIVILEGES IN SCHEMA auth_logs
    GRANT SELECT, INSERT ON TABLES TO dmama;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_logs
    GRANT USAGE, SELECT ON SEQUENCES TO dmama;
