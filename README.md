# hatchcert

Hatchcert is a config-driven tool to issue certificates using the ACME protocol.
It is intended to be easily deployable using configuration management systems
such as Ansible.
This tool is based on the [lego library](https://go-acme.github.io/lego/).


## Getting started

Create a configuration file, by default located in `/etc/hatchcert/config`:

    # Specify the ACME service to use
    # hatchcert will use Let's Encrypt production by default
    #acme_url https://acme-staging-v02.api.letsencrypt.org/directory

    # Confirm that you have read and accepted the terms of service of the ACME
    # service
    #accept_tos

    # Specify an email that will be used to contact you, e.g. when your
    # certificate is about to expire
    email hostmaster@example.com

    # Specify the root directory for the HTTP challenge solver
    webroot /run/acme

    # Specify domains to issue certificates for
    domain example.com

    # You can also request multiple names in one certificate
    domain example.net www.example.net

    # Optionally specify an executable file that will be called if, during
    # reconcile, a certificate was updated. This is typically used in order to
    # reload the certificates in various daemons. It is not called when you
    # forcefully issue certificates.
    #update-hook /etc/hatchcert/update-hook


Hatchcert is currently in development.
To get started:

* Create the appropriate configuration in `/etc/hatchcert/config`
* Run `hatchcert` once by hand, if you want to check if your configuration is
  valid; this will immediately register an account with the ACME server
* Optionally copy the update-hook script from `dist/update-hook` to `/etc/hatchcert/update-hook`
* Copy the cronjob from `dist/hatchcert.cron` to
  `/etc/cron.d/hatchcert` (adjust the cron schedule as required)



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
* Migrate away from Lego


## ACME DNS example

Basic support for Lego DNS challenge providers has been added.
The following example shows how to use the ACME DNS provider:

    # The provider is configured via environment variables
    env ACME_DNS_API_BASE=https://dnsauth.example.com/
    env ACME_DNS_STORAGE_PATH=/etc/hatchcert/acmedns.json
    dns acme-dns

    domain *.example.com

If you already have credentials for, you can place them in
`/etc/hatchcert/acmedns.json`:

   {"example.com": {
       "FullDomain": "04b30265-01ad-4275-88f2-3aaffe62d61e.dnsauth.example.com",
       "SubDomain": "04b30265-01ad-4275-88f2-3aaffe62d61e",
       "Username": "myusername",
       "Password": "justAnExample"
   }}

Notes:

* For wildcard certificates, the domain name in the `acmedns.json` config is
  without the wildcard
* The library Lego uses (goacmedns) has changed the format of the credentials
  file in a future version without backwards compatibility (lowercase keys)
* Future work will include a fully self-contained config without the secondary
  file


## nginx

To use the webroot challenge provider, create `/etc/nginx/snippets/acme.conf`
containing:

    # Let's encrypt
    location /.well-known/acme-challenge/ {
        alias /run/acme/.well-known/acme-challenge/;
    }

Include this snippet in every server block you want to issue certificates for.
For example:

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
