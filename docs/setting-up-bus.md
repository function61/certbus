Setting up the bus
==================

The event bus is backed by a project called EventHorizon.

TODO: some of the docs here should be moved to EventHorizon.

Contents:

- [Create DynamoDB table for EventHorizon](#create-dynamodb-table-for-eventhorizon)
- [Create IAM policy for EventHorizon](#create-iam-policy-for-eventhorizon)
- [Bootstrap EventHorizon](#bootstrap-eventhorizon)
- [Create tenant for CertBus](#create-tenant-for-certbus)
- [Create CertBus stream for your CertBus tenant](#create-certbus-stream-for-your-certbus-tenant)


Create DynamoDB table for EventHorizon
--------------------------------------

Create DynamoDB table in `eu-central-1` (currently the region is hardcoded..)
    * Name = `prod_eh_events`
    * Primary key = `s` (type: string)
    * Add sort key, name = `v` (type: number)


Create IAM policy for EventHorizon
----------------------------------

Create separate access keys for CertBus-manager and CertBus loadbalancers, add IAM permissions with following policy:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "dynamodb:GetItem",
                "dynamodb:PutItem",
                "dynamodb:DeleteItem",
                "dynamodb:Query"
            ],
            "Resource": [
                "arn:aws:dynamodb:*:*:table/prod_eh_events"
            ]
        }
    ]
}
```

NOTE: you should probably restrict write access from CertBus-loadbalancers (since they don't need it)


Bootstrap EventHorizon
----------------------

Run this to bootstrap EventHorizon:

```bash
$ certbus eh bootstrap
```

This created basic internal data structures for EventHorizon.


Create tenant for CertBus
-------------------------

EventHorizon is multi-tenanted system. CertBus uses different tenants to represent
different loadbalancer groups (maybe you have a production system and an intranet that you
want to keep separate configuration/certificates for).

That's all you need to know for now. You probably won't need that feature, and we'll call
you tenant 1. Create stream for tenant 1:

```bash
$ certbus eh stream-create / t-1
```


Create CertBus stream for your CertBus tenant
---------------------------------------------

Now create CertBus stream for tenant 1:

```bash
$ certbus eh stream-create /t-1 certbus
```
