# NULL

Use to throw away the file data.

```toml
[[filestores]]
name = "null"
type = "null"

# ...

[[sheets]]
name = "sheet1"
# ...
file_stores=["null"]
```

## Properties

The null provider supports the following properties:

* `name` (string) : [required] name of the filestore provider
* `type` (string) : [required] should be 'null'
* `log` (bool) : [optional] (false) should the file name be logged to the console
* `intermediate` (bool) : [optional] (false) should intermediate files be logged