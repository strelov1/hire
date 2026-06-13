-- Job geography (see openspec change ingest-job-geography): countries/regions and
-- a work_mode hint derived at ingest. These are SOURCE facts (from the structured
-- ATS fields and/or the parsed location string), distinct from the AI-derived
-- `enrichment` payload. countries/regions DEFAULT '{}' and work_mode DEFAULT ''
-- backfill existing rows as empty at migration time; the backfill command
-- (cmd/backfill-geo) then fills them from each row's stored location text, and
-- ingest keeps them fresh on every re-crawl.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS countries TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS regions   TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS work_mode TEXT   NOT NULL DEFAULT '';
