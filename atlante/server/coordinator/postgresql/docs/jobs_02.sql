ALTER TABLE IF EXISTS jobs
    ADD COLUMN style_name text DEFAULT '',
    ADD COLUMN style_location text DEFAULT '';

CREATE INDEX ON jobs (style_location);

