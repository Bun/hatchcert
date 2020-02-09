# hatchcert

Hatchcert is a config-driven tool to issue certificates using the ACME protocol.
It is intended to be easily deployable using configuration management systems
such as Ansible.
This tool is based around the [lego library](https://go-acme.github.io/lego/).


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


Hatchcert is currently in development.
To issue certificates, you must run `hatchcert` manually.


## Output and storage

The output directory structure produced by hatchcert is comparable to that of
other tools, such as acmetool:

* By default, all data is relative to `/var/lib/acme`
* Account information (including the private key) are written to `./account`
* Individual certificates are stored in `./certs/`
* A directory of symlinks pointing to the latest certificate is maintained in
  `./live/` for each (sub)domain


## TODO

* Helper tool to read and accept the terms of service
* Add DNS challenge provider support
* Private key permissions
* Packaging for Debian/Ubuntu
* Call to hook after refreshing certificates


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
