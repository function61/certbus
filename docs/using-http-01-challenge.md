Using HTTP-01 challenge
=======================

`DNS-01` challenge is great because:
- it supports wildcards
- doesn't need an active loadbalancer and
- works for intranets.

However you need `HTTP-01` if another domain (say, a customer) wants a CNAME to point to your
infrastructure, so you can't use DNS validation because you can't control the customer's DNS records.

(Sidenote: it might be possible to do DNS validation with CNAMEs, but
[LEGO didn't at least support it directly](https://github.com/go-acme/lego/issues/479).
I [even tried `LEGO_EXPERIMENTAL_CNAME_SUPPORT`](https://github.com/go-acme/lego/pull/791#issuecomment-461997317)
but it didn't seem to work.)

So if you decide to need `HTTP-01`, read on.


Mechanics
---------

CertBus supports writing `HTTP-01` challenges to an S3 bucket. It is assumed that you have your
loadbalancer configured to reverse proxy `http://ANY_HOSTNAME/.well-known/acme-challenge/...` from
that bucket. Edgerouter supports this.


AWS configuration
-----------------

### S3 bucket

Create S3 bucket to hold ACME challenges. You can also re-use an existing bucket (we use sane prefix
for challenges). The bucket needs to be publicly accessible to internet (ACME HTTP-01 challenges work
that way, so it's not inherently unsafe).

In our example our S3 bucket name is `my-supercool-bucket` and it's hosted in `eu-central-1`.

Create folder `acme-challenge` at root (this is
[strictly not necessary](https://stackoverflow.com/a/37847055) due to how S3 works), but is semantically
great so you won't come back in 5 years and wonder what this empty bucket is used for..


### Auto-delete

You might want to set up ACME challenge files to be automatically deleted in one day. Our code cleans up
the temporary challenge files, but in case the deletion would fail, they should be cleaned up.

I did these steps:

- Create lifecycle rule
- Rule name = Delete ACME challenges after 1 day
- Prefix = `acme-challenge/`
- Action = expire current versions of objects
- Number of days = 1


### IAM permissions

CertBus already uses AWS (due to EventHorizon). We'll use our existing authentication
(ENV vars `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) to give CertBus permission to write ACME
challenges to S3 bucket.

Head over to IAM, create policy with this JSON:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "writeACMEchallenges",
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:PutObjectAcl"
            ],
            "Resource": [
                "arn:aws:s3:::my-supercool-bucket/acme-challenge/*"
            ]
        }
    ]
}
```

I named the policy `WriteACMEchallenges`. Attach it to the user and/or role (role if using Lambda)
that CertBus runs under.


CertBus configuration
---------------------

First print out CertBus configuration. We need to add S3 config there:

```console
$ certbus conf display
```

You'll get this output:

```json
{
    "lets_encrypt": {},
    "cloudflare_credentials": {},
    "kek_public_key": "..."
}
```

Save that to `config.json`. Add `acme_http01_challenges` config:

```json
{
    "lets_encrypt": {},
    "cloudflare_credentials": {},
    "kek_public_key": "...",
    "acme_http01_challenges": {
        "bucket": "my-supercool-bucket",
        "region": "eu-central-1"
    }
}
```

Now update it back:

```console
$ certbus conf update < config.json
```

Delete `config.json` (for security purposes).


Loadbalancer configuration
--------------------------

This one varies by loadbalancer. Our config for Edgerouter looks like this:

```json
{
  "id": "acme-validations",
  "frontends": [
    {
      "kind": "path_prefix",
      "path_prefix": "/.well-known/acme-challenge/",
      "strip_path_prefix": true,
      "allow_insecure_http": true
    }
  ],
  "backend": {
    "kind": "reverse_proxy",
    "reverse_proxy_opts": {
      "origins": [
        "https://s3.eu-central-1.amazonaws.com/my-supercool-bucket/acme-challenge"
      ]
    }
  }
}
```

NOTE: our config first translates `/.well-known/acme-challenge/TOKEN` into `/token`, then prefixes
that with the reverse proxy origin, i.e. resulting URL will be
`https://s3.eu-central-1.amazonaws.com/my-supercool-bucket/acme-challenge/token`.

