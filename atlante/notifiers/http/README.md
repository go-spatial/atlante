# http notifier

This notifier will post status updates to the configured http end point.

```toml

...

[notifier]
    type = "http"
    url_template="http://localhost:9090/jobs/{{.JobID}}/status"

```

The http notififer will post json status update messages to the url provided by the 
`url_template` config value. The only config options is the `.JobID`. Each instance of `{{.JobID}}` will be replaced with the job id.

# Properties

The provider supports the following properties

* `type`         (string) : [required] should always be 'http'
* `url_template` (string) : [required] the url to post to. Instance of `{{.JobID}}` will be replaced with the job id.

# Posted JSON:

The url will receive a POST with the following JSON:

```js
{
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
}
```
