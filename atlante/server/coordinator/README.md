# Coordinator

The job of the coordinator is to act as a queue for jobs.
The coordinator has two important jobs:

1. Make sure jobs that are the same receive the same JobID and are
not requeued into the system.
2. Provide information about Jobs that are currently in the system.

The coordinator does not enqueue a job just create a new job or return
an already existing job based on the information given to it.
