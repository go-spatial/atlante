#Configuration of Atlante

## Webserver

```toml

[webserver]

    port = "9090"

```

### Properties

The webserver has the following properties

* `hostname` (string) : [optional] ("") the hostname
* `port` (string) : [optional] (":8080") the port
* `scheme` (string) : [optional] ("http") the scheme to use
* `webserver.headers` (table) : [optional] additional headers to add to each response
* `webserver.queue` (table) : [optional] the queue to use to send jobs to workers

## Sheets

```toml

[[providers]]
...

[[filestores]]
...

[[sheets]]

   name = "50k"
   provider_grid = "postgistDB50k"
   style = "files://...."
   template = "templates/...."
   file_stores = [ "file", "s3" ]

```
### Properties

* `name` (string) : [required] the name of the sheet
* `template` (string) : [required] the template to use 
* `style` (string) : [required] the style to use for this sheet
* `description` (string) : [optional] ("") the description for this sheet
* `provider_grid` (string) : [required] the name of the grid provider
* `file_stores` (string) : [required] the names of file stores to use
* `dpi` (int) : [optional] (144) the DPI to use

