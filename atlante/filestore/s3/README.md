# s3

Use to copy artificates to s3 buckets

```toml
[[filestores]]
name = "s3"
type = "s3"
base_path="/"
bucket="final"
intermediate=true
intermediate_bucket="intermediate"


# ...

[[sheets]]
name = "sheet1"
# ...
file_stores=["s3"]
```

## Properties

The file supports the following properties:

* `name` (string) : [required] name of the filestore provider
* `type` (string) : [required] should be 's3'
* `bucket` (srint) : [required] bucket to put files into 
* `base_path` (string) : [optional] ("")a base path to add to files being put
* `group` (bool) : [optional] (false) should the files be grouped into a directory by the group name (mbgid or latlng).
* `intermediate` (bool) : [optional] (false) should the provider copy over intermediate files.
* `intermediate_bucket` (string) : [optional] (bucket value) the bucket to put intermediate files into
* `intermediate_base_path` (string) : [optional] (base_path) the base pat to add to intermediate files
* `region` (string) : [optional] ("us-east-1") the aws region
* `end_point` (string) : [optional] ("") the aws end point
* `aws_access_key_id` (string) [optional] aws key
* `aws_secret_access_key` (string) [optional] aws secret key
