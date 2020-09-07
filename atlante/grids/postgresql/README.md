# postgresql

A grid provider that can connect to a Postgres Database.

```toml

[[providers]]
    name = "PostgisDB50K"
    type = "postresql"
    host = "docker.for.mac.localhost"
    database = "50k"
    user = "postgis"
    password = "password"
    scale = 50000

```

# Properties

The provider supports the following properties

* `host`     (string) : [required] the database host
* `name`     (string) : [required] the name of the provider (this will be normalized to the lowercase)
* `password` (string) : [required] the user password
* `port`     (string) : [required] the database post
* `scale`    (number) : [required] The scale of the grid in meters, e.g. 5000, 50000, 250000
* `type`     (string) : [required] should be 'grid5k'
* `user`     (string) : [required] the user for the database

* `edit_data_format` (string) : [optional] (RFC3339) the data format of the data values in the database
* `edit_by`          (string) : [optional] ("") if edit_by is not provided default value to use
* `max_connections`  (number) : [optional] the max number of connections to keep in the pool
* `srid`             (number) : [optional] (3857) the srid of the data (not used; currently)
* `ssl_mode`         (string) : [optional] the ssl mode for postgres SSL
* `ssl_key`          (string) : [optional] the ssl key for postgres SSL
* `ssl_cert`         (string) : [optional] the ssl cert for postgres SSL
* `ssl_root_cert`    (string) : [optional] the ssl root cert

## SQL Properties

These properties allow one to redefine the sql used to retrieve the grid
There are three, for mdgid, for lng/lat and bounds values
* `query_mdgid` (string) : [optional] the sql used to retrieve the grid values for an mdgid

Example SQL for a 50k grid

```sql
SELECT
  mdg_id,
  sheet,
  series,
  nrn,

  swlat_dms,
  swlon_dms AS swlng_dms,
  nelat_dms,
  nelon_dms AS nelng_dms,

  swlat,
  swlon AS swlng,
  nelat,
  nelon AS nelng,

  country,
  last_edite AS edited_by,
  last_edi_1 AS edited_at
FROM
  grids.grid50K
WHERE
	mdg_id = $1
LIMIT 1;
`
```

** `$1` is the mdgid value

* `query_lnglat` (string) : [optional] the sql used to retrieve the grid values for an lng/lat pair

Example SQL for a 50k grid

```sql
SELECT
  mdg_id,
  sheet,
  series,
  nrn,

  swlat_dms,
  swlon_dms AS swlng_dms,
  nelat_dms,
  nelon_dms AS nelng_dms,

  swlat,
  swlon AS swlng,
  nelat,
  nelon AS nelng,

  country,
  last_edite AS edited_by,
  last_edi_1 AS edited_at
FROM
  grids.grid50K
WHERE
  ST_Intersects(
    wkb_geometry,
    ST_Transform(
      ST_SetSRID(
        ST_MakePoint($1,$2),
        $3
      ),
      4326
    )
  )
LIMIT 1;
```
** `$1` is the Lng value

** `$2` is the lat value

** `$3` is the srid

* `query_bounds` (string) : [optional] the sql used to retrieve the grid values for an mdgid

Example SQL for a 50k grid

```sql
		SELECT
		  mdg_id,
		  sheet,
		  series,
		  nrn,
      swlat,
      swlon AS swlng,
      nelat,
      nelon AS nelng,
		  country,
		  last_edite AS edited_by,
		  last_edi_1 AS edited_at
		FROM
		  grids.grid50K
		WHERE
		  ST_Intersects(
		    wkb_geometry,
		    ST_Transform(
		      ST_SetSRID(
		        ST_MakeEnvelope($1,$2,$3,$4),
		        $5
		      ),
		      4326
		    )
		  )
		LIMIT 1;
```

** `$1`,`$2`,`$3`, and `$4` are the bounds values
** `$5` is the srid

# Expected Table Layout:

If the sql's arn't provided then the following assumptions are made.

The provider expects there to be at database called grids.
The `grids` database should have a table called `grid50k`, with the 
following columns:
`mdg_id`, `sheet`, `series`, `nrn`, `swlat_dms`, `swlon_dms`, `nelat_dms`, `nelon_dms`,
`swlat`, `swlon`, `nelat`, `nelon`, `country`, `last_edite`, `last_edi_1`,`wkb_geometry`

---

## PostGIS notes for `atlante`

The following notes regarding the PostGIS database assume the following `providers` setup.

``` toml
# config.toml
[[providers]]
  name     = "the_database"
  type     = "postgresql"
  database = "grids"
  user     = "tegola"
  scale    = 50000
```

### Setup the Database

From the shell console

``` bash
# The provider expects there to be at database called grids.
createdb grids
# enter the database to create the TABLE, etc
psql grids
```

### Create Tables

From the SQL command line `grids=# `.

``` sql
-- Enabling PostGIS, From http://postgis.net/install/
CREATE EXTENSION postgis;  -- Enable PostGIS (includes raster)
CREATE EXTENSION postgis_topology;  -- Enable Topology

-- CREATE this TABLE
--   comments include sample data
CREATE TABLE grid50k (
  mdg_id      varchar(80),    -- "MDG_ID":"__________"
  sheet       varchar(80),    -- "SHEET":"_____"
  series      varchar(80),    -- "SERIES":"____"
  nrn         varchar(80),    -- "NRN":"__________"
  swlat_dms   varchar(80),    -- "SWLAT_DMS":"32°30\"0.0'N"
  swlon_dms   varchar(80),    -- "SWLON_DMS":"117°15\"0.0'W"
  nelat_dms   varchar(80),    -- "NELAT_DMS":"32°45\"0.0'N"
  nelon_dms   varchar(80),    -- "NELON_DMS":"117°0\"0.0'W"
  swlat       real,           -- "SWLAT":32.5
  swlon       real,           -- "SWLAT":32.5
  nelat       real,           -- "NELAT":32.75000000000013
  nelon       real,           -- "NELON":-116.99999999999996
  country     varchar(80),    -- "COUNTRY":"United States"
  last_edite  varchar(80),    -- "LAST_EDITE":"couldBeUser_or???"
  last_edi_1  varchar(80)     -- "LAST_EDI_1":"2018-07-09T00:00:00.000"
);

-- Schema and permissions
-- Recall the settings from config.toml
--   database = "grids"
--   user     = "tegola"
CREATE SCHEMA grids;
GRANT USAGE ON SCHEMA grids TO tegola;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA grids TO tegola;
SELECT * INTO grids.grid50k FROM grid50k WHERE 1=1;  -- TODO Could be refactored
```

### Add Data from `.gdb`

From the shell console

``` bash
# Convert GDB to Shapefile
mkdir shape  # for output of `ogr2ogr`
ogr2ogr -f "ESRI Shapefile" shape/data.shp data.gdb/

# Convert Shapefile to SQL into table `grid50k`
shp2pgsql shape/data.shp grid50k > data.sql

# Put SQL into PostgreSQL via CLI
psql -d grids -U tegola -f data.sql

# To test, echo back some data from the SCHEMA.TABLE
psql grids --command="SELECT country    FROM grids.grid50k LIMIT 10;"
psql grids --command="SELECT nrn        FROM grids.grid50k LIMIT 10;"
psql grids --command="SELECT sheet      FROM grids.grid50k LIMIT 10;"
psql grids --command="SELECT last_edite FROM grids.grid50k LIMIT 10;"
psql grids --command="SELECT last_edi_1 FROM grids.grid50k LIMIT 10;"
```

