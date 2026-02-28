# Domains and TLS

This document defines DNS and certificate strategy for production and development domains.

## DNS strategy

Production:
- `hemis.domain.com` -> VPS public IP
- `tiap.domain.com` -> VPS public IP

Development:
- `*.dev.domain.com` -> VPS public IP (wildcard)

Wildcard DNS is recommended to avoid per-developer manual DNS changes.

## TLS strategy

Production:
- Let’s Encrypt certificates per domain via HTTP-01 challenge
- Challenge served through proxy on port 80:
  - `/.well-known/acme-challenge/`

Development:
- Wildcard certificate for `*.dev.domain.com` via DNS-01 challenge
- Recommended provider automation: Cloudflare API token

Reason: avoid Let’s Encrypt rate limits and avoid per-dev cert issuance.

## Certificate storage

Proxy stack stores certificates via mounted volumes:
- `/etc/letsencrypt` in proxy and certbot containers
- webroot challenge directory mounted at `/var/www/certbot`

Certificates and private keys must not be committed to Git.

## Renewal

- Certbot runs on a schedule (e.g., every 12 hours) and renews certificates as needed.
- After renewal, the proxy must pick up updated certificates. If using shared volumes, a reload is typically sufficient.

## Operational notes

- DNS-01 wildcard requires access to DNS provider API.
- Treat the DNS API token as a secret:
  - store in `.env` files not committed
  - restrict permissions to admin only
- Consider certificate rotation procedures and documentation if multiple admins exist.

