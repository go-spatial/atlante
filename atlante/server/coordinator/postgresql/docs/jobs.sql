CREATE EXTENSION postgis;

CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    mdgid TEXT NOT NULL,
    sheet_number integer DEFAULT 0,
    sheet_name TEXT NOT NULL,
    queue_id TEXT,
    job_data TEXT,
    bounds GEOMETRY,
    created TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX ON jobs (mdgid);

CREATE INDEX ON jobs (sheet_number);

CREATE INDEX ON jobs (sheet_name);

CREATE INDEX ON jobs (queue_id);

CREATE INDEX bounds_polygon_idx ON bounds USING GIST (bounds);


CREATE TABLE statuses (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    description TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE default NOW()
);


CREATE INDEX ON statuses (job_id);

-- Find job by job id
SELECT 
    job.mdgid,
    job.sheet_number,
    job.sheet_name,
    job.queue_id,
    job.created as enqueued,
    jobstatus.status,
    jobstatus.description,
    jobstatus.created as updated
FROM jobs AS job
JOIN statuses AS jobstatus ON job.id = jobstatus.job_id
WHERE job.id = $1
ORDER BY jobstatus.id desc limit 1;
