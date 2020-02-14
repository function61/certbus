Setting up the bus
==================

The event bus is backed by a project called EventHorizon.

Contents:

- [Set up EventHorizon & AWS IAM users for CertBus](#set-up-eventhorizon-aws-iam-users-for-certbus)
- [Create tenant for CertBus](#create-tenant-for-certbus)
- [Create CertBus stream for your CertBus tenant](#create-certbus-stream-for-your-certbus-tenant)


Set up EventHorizon & AWS IAM users for CertBus
-----------------------------------------------

You need to set up EventHorizon:
[Setting up data storage](https://github.com/function61/eventhorizon/blob/master/docs/setting-up-data-storage/README.md).

(NOTE: before you do the above, read the below instructions first - this makes sense later)

Now you should add two IAM users for CertBus (they both get their own API keys):

- CertBus-manager
  * Attach to user group `EventHorizon-readwrite`
- CertBus-client
  * Attach to user group `EventHorizon-read` (clients don't need to write on the bus)

After you create the users, take note of the API credentials - you'll need them.


Create tenant for CertBus
-------------------------

Like in the previous section's EventHorizon guide, you need to have AWS credentials as ENV vars.

EventHorizon is multi-tenanted system. CertBus uses different tenants to represent
different loadbalancer groups - maybe you have a production system and an intranet that you
want to keep separate configuration/certificates for.

That's all you need to know for now. You probably won't need that feature, and we'll call
your first loadbalancer group tenant 1. Create stream for tenant 1:

```console
$ certbus eh-prod stream-create /t-1
```


Create CertBus stream for your CertBus tenant
---------------------------------------------

Now create CertBus stream for tenant 1:

```console
$ certbus eh-prod stream-create /t-1/certbus
```
