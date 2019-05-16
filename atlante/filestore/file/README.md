# File

Use to copy artificates to a different location on the file system.

```toml
[[filestores]]
name = "usr/dir"
type = "file"
base_path="/usr/dir"

# ...

[[sheets]]
name = "sheet1"
# ...
file_stores=["usr/dir"]
```

## Properties

The file supports the following properties:

* `name` (string) : [required] name of the filestore provider
* `base_path` (string) : [required] the location on the file system to write the files to
* `type` (string) : [required] should be 'file'
* `group` (bool) : [optional] (false) should the files be grouped into a directory by the group name (mbgid or latlng).
* `intermediate` (bool) : [optional] (false) should the provider copy over intermediate files.