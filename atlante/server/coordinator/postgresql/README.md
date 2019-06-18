# postgresql

A coorinator provider that can connect to a Postgres Database.

```toml

[[webserver.coordinator]]
    type = "postresql"
    host = "docker.for.mac.localhost"
    database = "management"
    user = "postgis"
    password = "password"

```

# Properties

The provider supports the following properties

* `type` (string) : [required] should be 'grid5k'
* `name` (string) : [required] the name of the provider (this will be normalized to the lowercase)
* `host` (string) : [required] the database host
* `port` (string) : [required] the database post
* `user` (string) : [required] the user for the database
* `password` (string) : [required] the user password
* `ssl_mode` (string) : [optional] the ssl mode for postgres SSL
* `ssl_key` (string) : [optional] the ssl key for postgres SSL
* `ssl_cert` (string) : [optional] the ssl cert for postgres SSL
* `ssl_root_cert` (string) : [optional] the ssl root cert
* `max_connections` (number) : [optional] the max number of connections to keep in the pool

## SQL options.
The following options are used to specify the sql that is used to manage the 
database. (note this can be brittle.)

* `query_new_job` (string): the sql is run to create a new job.

Default SQL:

```sql

INSERT INTO jobs(
	mdgid,
	sheet_number,
	sheet_name,
	bounds
)
VALUES($1,$2,$3,ST_GeometryFromText($4, $5))
RETURNING id;

```

    * $1 will be the mdgid (string)
    * $2 will be a sheet number (uint32)
    * $3 will be the sheet name (string)
    * $4 will be wkt of the bounds of the grid
    * $5 will be the srid -- hardcode to 4326 for now

* `query_update_queue_job_id` (string): the sql is run to update the queue job id.
Default SQL:

```sql

UPDATE jobs 
SET queue_id=$2
WHERE id=$1
	
```

    * $1 will be the jobid (int)
    * $2 will be the queue_id (string)

* `query_update_job_data` (string): the sql is run to update the job data
Default SQL:

```sql

UPDATE jobs 
SET job_data=$2
WHERE id=$1

```
    * $1 will be the jobid (int)
    * $2 will be the job_data (string)


* `query_insert_status` (string): the sql is run to insert a new status for a job

```sql

INSERT INTO statuses(
	job_id,
	status,
	description
)
VALUES($1,$2,$3);
	
```
    * $1 will be the jobid (int)
    * $2 will be the status (string)
    * $3 will be the description (string)

* `query_select_job_id` (string): the sql is used to find job for a job_id

```sql
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
```
    * $1 will be the jobid (int)

    The list order is the order in which the items need to occure.
    The system is expect the sql to return zero or one row only.


* `query_select_mdgid_sheetname` (string): the sql is used to find jobs for an mdgid/sheetname 

```sql
SELECT 
    job.id,
    job.queue_id,
    job.created as enqueued,
    jobstatus.status,
    jobstatus.description,
    jobstatus.created as updated
FROM jobs AS job
JOIN statuses AS jobstatus ON job.id = jobstatus.job_id
WHERE job.mdgid = $1 AND job.sheet_number = $2 AND job.sheet_name = $3
ORDER BY jobstatus.id desc limit 1;
```

    * $1 will be the mdgid (string)
    * $2 will be the sheet number (int)
    * $3 will be the sheet name (string)

    The list order is the order in which the items need to occure.
    The system is expect the sql to return zero or one row only.

Create sqls for the original tables can be found in the [docs/jobs.sql folder.](doc/jobs.sql)
