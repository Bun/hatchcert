# hatchcert

Hatchcert is a config-driven tool to issue certificates using the ACME protocol.
It is intended to be easily deployable using configuration management systems
such as Ansible.
This tool is based on [acmez](https://github.com/mholt/acmez).


## Getting started

Build hatchcert using `make hatchcert` or create a deb file using `make deb`.

Create a configuration file, by default located in `/etc/hatchcert/config`:

    # Required: Confirm that you have read and accepted the terms of service of the
    # ACME service
    #accept_tos

    # Required: Specify an email that will be used to contact you
    #email hostmaster@example.com

    # Optional: Specify the ACME service to use
    # hatchcert will use Let's Encrypt production by default
    # Also accepted: prod | staging | pebble
    #acme_url https://acme-staging-v02.api.letsencrypt.org/directory

    # Optional: Specify which certificate profile to use.
    # See https://letsencrypt.org/docs/profiles/ for more information
    #profile shortlived

    # Optional: specify which chain you prefer; pick the common name of an
    # intermediary or root certificate.
    # See https://letsencrypt.org/certificates/ for more information
    #preferred-chain ISRG Root X1


    #
    ## Challenge solvers
    #

    # Currently HTTP-01 is the only supported challenge method. Specificy a method
    # to solve it:

    # Use built-in HTTP server (listens on :80 by default; specify a different
    # address if needed); note that hatchcert may need elevated permissions to
    # listen on low ports!
    #http

    # Have an existing webserver serve this directory under
    # "/.well-known/acme-challenge/" for each domain
    #httpdir /run/acme


    #
    ## Certificates
    #

    # Specify domains to issue certificates for
    #domain example.com

    # You can also request multiple names in one certificate
    #domain example.net www.example.net


    #
    ## Hooks
    #

    # Optionally specify an executable file that will be called if, during
    # reconcile, a certificate was updated. This is typically used in order to
    # reload the certificates in various daemons. It is not called when you
    # forcefully issue certificates.
    #update-hook /etc/hatchcert/update-hook


While Hatchcert is somewhat of an "in development" project, I've been using it
as sole solution to issue LE certificates on my own infrastructure since early 2020.
To get started:

* Create the appropriate configuration in `/etc/hatchcert/config`
* Run `hatchcert` once by hand if you want to check if your configuration is
  valid; this will immediately register an account with the ACME server
* Optionally copy the update-hook script from `dist/update-hook` to `/etc/hatchcert/update-hook`
* Enable the systemd timer with `systemctl enable --now hatchcert.timer` (not using
  the deb version? `cp dist/hatchcert.{service,timer} /ur/lib/systemd/system`)



## Running hatchcert as non-root user

Running hatchcert as root is the easiest option, but not strictly required:

* Create the `/var/lib/acme` directory in such a way that your desired user can
  write to it (or specify an alternative base path using the `-path` parameter)
* Modify the cronjob so that hatchcert runs as the desired user
* Ensure your update-hook script runs in such a way that it can reload services
  (for example, with the appropriate sudo configuration)


## Output and storage

The output directory structure produced by hatchcert is comparable to that of
other tools, such as acmetool:

* By default, all data is stored relative to `/var/lib/acme`
* Account information (including the private key) is written to `./account`
* Individual certificates are stored in `./certs/`
* A directory of symlinks pointing to the latest certificate is maintained in
  `./live/` for each (sub)domain


## TODO

* Helper tool to read and accept the terms of service
* Private key permissions


## ACME challenge solvers


### HTTP (http-01) using the built-in server

To use the built-in webserver (for example, if you're not already running a
webserver), use the `http` keyword in the configuration.
Specify a bind address with `http 10.1.2.3:80` or the default port 80 will be
used.


### HTTP (http-01) using an external webserver

Hatchcert can write HTTP challenge files to a directory using the `httpdir`
keyword.

For example, to use the webroot challenge provider with nginx with
`httpdir /run/acme`, create `/etc/nginx/snippets/acme.conf` containing:

    # Let's encrypt
    location /.well-known/acme-challenge/ {
        alias /run/acme/;
    }

Then, include this snippet in every server block you want to issue certificates
for.  For example:

    server {
        server_name example.com;
        listen 0.0.0.0:443 ssl http2;
        listen [::]:443 ssl http2;
        add_header Strict-Transport-Security max-age=31536000;

        include snippets/acme.conf;
        ssl_certificate /var/lib/acme/live/example.com/fullchain;
        ssl_certificate_key /var/lib/acme/live/example.com/privkey;

        root /var/www/example.com/htdocs;
    }

Deprecation notice: A legacy keyword `webroot` exists that creates a
subdirectory called `.well-known/acme-challenge/` relative to the configured
path. This behavior was undesired and is no longer supported. You can
replicate the behavior by simply appending `.well-known/acme-challenge` to the
`httpdir` keyword.


### DNS (dns-01)

Due to the lack of support for a general DNS updating protocol supported by most
DNS providers, DNS-01 is only supported through RFC 2136 support.

An example configuration may look like:

    # dns rfc2136 <nameserver> <optional:zone>
    dns rfc2136 ns.example.com
    # tsig <keyname> <secret> <optional:algo>
    tsig updatekey c2VjcmV0LXRoaXMtaXMtbm90LXNlY3VyZQ==
    domain example.com www.example.com

The `domain` keyword will use the last `dns` and `tsig` values configured,
allowing multiple independent domains. If any domains do not require `tsig`
(not recommended), configure them before any that do.
If you do not specify the zone name, it will be derived from the nearest SOA of
the domains being issued.


### Zero-configuration DNS (dns-persist-01)

This DNS challenge requires persistent DNS record, which means that no DNS
entries need to be made during normal Hatchcert operation.
See: https://letsencrypt.org/2026/02/18/dns-persist-01

An example configuration may look like:

    dns persist
    domain example.org

You can obtain the necessary account URL through the `hatchcert account`
command or by inspecting `/var/lib/acme/account`.
