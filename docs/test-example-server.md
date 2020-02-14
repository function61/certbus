Example server
==============

Your loadbalancer will need access to SSL certificates from CertBus. This is an example
server with which you can verify that CertBus is working.

Contents:

- [Start the server](#start-the-server)
- [Test connectivity with curl](#test-connectivity-with-curl)


Start the server
----------------

```console
$ certbus example-server
```


Test connectivity with curl
---------------------------

We assume that you already issued a certificate under your domain to `foo.example.com`.

Now that example server is running, from another terminal test connecting to the demo
server running on localhost:

```console
$ curl --resolve foo.example.com:443:127.0.0.1 https://foo.example.com/
```

(You need the `--resolve` switch if you want to make curl believe that `foo.example.com` resolves to `localhost`)

You'll get:

- `greetings from /` if everything works => a 404 (the server doesn't serve anything).
- TLS error if the certificate was not found.
    * If you're unsure if this is TLS error add `--insecure` to bypass cert validation.

While the example server is running, you can now test issuing and removing certificates from
CertBus-manager. The changes should propagate to your example server.
