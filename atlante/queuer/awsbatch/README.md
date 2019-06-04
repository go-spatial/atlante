# awsbatch

Use to copy artificates to s3 buckets

```toml
[webserver.queue]
	type="awsbatch"
	aws_access_key_id = "XXXXXXXXXXXXXXXXXXXXXXX"
	aws_secret_access_key = "XXXXXXXXXXXXXXXXXXXXXXX"
	region="us-west-1"
	job_queue = "queue"
	job_definition = "def"
	job_object_key = "job-data"

```

## Properties

The file supports the following properties:

* `type` (string) : [required] should be 'awsbatch'
* `job_queue` (string) : [required] the queue name 
* `job_definition` (string) : [required] the job definition on aws
* `job_parameters` (table) : [optional] static parameters to assign to the job.
* `job_object_key` (string) : [optional] ("job-data") the parameter field to fill with the job data. This will overwite that job_parameters field if it is set there as well.
* `region` (string) : [optional] ("us-east-1") the aws region
* `end_point` (string) : [optional] ("") the aws end point
* `aws_access_key_id` (string) [optional] aws key
* `aws_secret_access_key` (string) [optional] aws secret key

## Credential chain

If the `aws_access_key_id` and `aws_secret_access_key` are not set, then the [credential provider chain](http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) will be used. The provider chain supports multiple methods for passing credentials, one of which is setting environment variables.
