CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE jobs (
    id serial PRIMARY KEY,
    mdgid text NOT NULL,
    sheet_number integer DEFAULT 0,
    sheet_name text NOT NULL,
    queue_id text,
    job_data text,
    bounds geometry(polygon, 4326) NOT NULL,
    style_name text DEFAULT '',
    style_location text DEFAULT '',
    created timestamp WITH time zone DEFAULT NOW()
);

CREATE INDEX ON jobs (mdgid);

CREATE INDEX ON jobs (sheet_number);

CREATE INDEX ON jobs (sheet_name);

CREATE INDEX ON jobs (queue_id);

CREATE INDEX ON jobs (style_location);

CREATE INDEX bounds_polygon_idx ON jobs USING GIST (bounds);

CREATE TABLE job_statuses (
    id serial PRIMARY KEY,
    job_id integer NOT NULL,
    status text NOT NULL,
    description text NOT NULL,
    created timestamp WITH time zone DEFAULT NOW()
);

CREATE INDEX ON job_statuses (job_id);

-- Find job by job id
--
-- SELECT
--     job.mdgid,
--     job.sheet_number,
--     job.sheet_name,
--     job.style_location,
--     job.queue_id,
--     job.created AS enqueued,
--     jobstatus.status,
--     jobstatus.description,
--     jobstatus.created AS updated
-- FROM
--     jobs AS job
--     JOIN job_statuses AS jobstatus ON job.id = jobstatus.job_id
-- WHERE
--     job.id = $1
-- ORDER BY
--     jobstatus.id DESC
-- LIMIT 1;
-- 

