Configuring manager
===================

Contents:

- [Where to run](#where-to-run)
- [Create private key for manager](#create-private-key-for-manager)
- [Create private key for loadbalancer group](#create-private-key-for-loadbalancer-group)
- [Create manager's configuration](#create-managers-configuration)
- [Testing that configuration is readable](#testing-that-configuration-is-readable)
- [Why store the configuration on the bus?](#why-store-the-configuration-on-the-bus)


Where to run
------------

You can run CertBus-manager on:

- AWS Lambda
- On premises
    * it's just a program you can run manually from command line or from cron


Create private key for manager
------------------------------

This private key is used to decrypt the sensitive configuration that is stored in an
encrypted form on the bus. Loadbalancer can see "config change" events, but cannot decrypt
the config values.

Create `certbus-manager.key` RSA key:

```console
$ openssl genrsa -out certbus-manager.key 4096
```

(there's no manager.pub because we don't need it)


Create private key for loadbalancer group
-------------------------------------

Note: your CertBus manager doesn't need (and should not know) the loadbalancer's private key,
so you can generate the key somewhere else and just enter the public key into CertBus-manager.

Generate private key:

```console
$ openssl genrsa -out loadbalancer.key 4096
```

Now, extract its public key:

```console
$ openssl rsa -in loadbalancer.key -outform PEM -pubout -RSAPublicKey_out -out loadbalancer.pub
```


Create manager's configuration
------------------------------

From this point on, you will have to have have the AWS credentials as ENV variables defined
(as advised in the bus set-up tutorial). You now also have to define:

```console
$ export EVENTHORIZON_TENANT=prod:1
```

(`1` is tenant #1 - it's just the default tenant number which you don't need to customize
unless you're running in multi-tenant mode)

This configuration will contain:

- LetsEncrypt credentials
- DNS provider credentials (we only support Cloudflare for now)
- Loadbalancer group's public key

Create `config-temp.json` (temporary as a file because the config will live on the bus) file with content:

```javascript
{
    "lets_encrypt": {
        "email": "...",
        "private_key": "-----BEGIN EC PRIVATE KEY-----\n...\n-----END EC PRIVATE KEY-----\n",
        "registration": {
            "body": {
                "status": "valid",
                "contact": [
                    "..."
                ]
            },
            "uri": "https://acme-v02.api.letsencrypt.org/acme/acct/..."
        }
    },
    "cloudflare_credentials": {
        "email": "...",
        "api_key": "..."
    },
    "kek_public_key": "-----BEGIN RSA PUBLIC KEY-----\n...\n-----END RSA PUBLIC KEY-----\n"
}

```

NOTE: `kek_public_key` is your `loadbalancer.pub` content (**and NOT the manager's**)

Then upload this to the bus:

```
$ certbus conf update < config-temp.json
```

Test (`conf-display`) that the config is readable, and then as cleanup remove the temp file
so sensitive config is not in filesystem:

```console
$ rm config-temp.json
```

We're finished configuring the manager!


Testing that configuration is readable
--------------------------------------

Run this:

```console
$ certbus conf display
```


Why store the configuration on the bus?
---------------------------------------

Why not? By knowing the manager's private key, you can run the manager from anywhere
because you have access to the configuration from anywhere. Use manager at the same time
from Lambda and from your own computer..

Example use case: Lambda can handle renewals, and you can add/remove certificates from
your computer.
