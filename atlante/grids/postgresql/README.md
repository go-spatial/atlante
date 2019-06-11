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
* `srid` (number) :  [optional] (3857) the srid of the data (not used; currently)
* `edit_data_format` (string): [optional] (RFC3339) the data format of the data values in the database
* `edit_by` (string) :[optional] ("") if edit_by is not provided default value to use


# Expected Table Layout:

The provider expectes there to be at database called grids.
The `grids` database should have a table called `grid50k`, with the 
following columns:
`mdg_id`, `sheet`, `series`, `nrn`, `swlat_dms`, `swlon_dms`, `nelat_dms`, `nelon_dms`,
`sw_lat`, `swlon`, `nelat`, `nelon`, `country`, `last_edite`, `last_edi_1`

