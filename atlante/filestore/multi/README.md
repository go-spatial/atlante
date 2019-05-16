# MULTI

Used to group a set of other file stores.

There should be no need to use this filestore. This is here
for internal use only. Sheets can take multiple file stores
, and that should be used instead. 

```toml
[[filestores]]
name = "multi"
type = "multi"
file_stores = [ 'usr/local/', 'null']

# ...

[[sheets]]
name = "sheet1"
# ...
file_stores=["multi"]
```

## Properties

The multi provider supports the following properties:

* `name` (string) : [required] name of the filestore provider
* `type` (string) : [required] should be 'multi'
* `file_stores` (bool) : [required] list of other files stores