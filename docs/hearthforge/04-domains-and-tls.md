# Domains and TLS

This document defines DNS and certificate strategy for production and development domains.

## DNS Strategy

Production:
```
hemis.domain.com  →  VPS public IP
tiap.domain.com   →  VPS public IP
```

Development:
```
*.dev.domain.com  →  VPS public IP  (wildcard)
```

Wildcard DNS is recommended to avoid per-developer manual DNS changes.

## TLS Strategy

**Production:**
- Let's Encrypt certificates per domain via HTTP-01 challenge
- Challenge served through proxy on port 80 at `/.well-known/acme-challenge/`

**Development:**
- Wildcard certificate for `*.dev.domain.com` via DNS-01 challenge
- Recommended provider: Cloudflare API token automation

Reason: avoid Let's Encrypt rate limits and avoid per-dev cert issuance.

When SmeltForge is installed (the standard setup), Caddy handles TLS automatically for both production and dev preview domains via Let's Encrypt DNS-01 challenge. No manual certbot setup is needed.

The manual certbot setup described below applies only when running HearthForge without SmeltForge, using a standalone Nginx proxy.

## Certificate Storage (Caddy, standard setup)

Caddy manages certificate storage automatically. Certificates are stored in Caddy's data directory and renewed without admin intervention. No additional configuration is required.

## Certificate Storage (Nginx, standalone setup)

Proxy stack stores certificates via mounted volumes:
- `/etc/letsencrypt` in proxy and certbot containers
- Webroot challenge directory mounted at `/var/www/certbot`

Certificates and private keys must not be committed to Git.

## Renewal

Certbot runs on a schedule (e.g. every 12 hours) and renews certificates as needed. After renewal, the proxy must pick up updated certificates. If using shared volumes, a reload is typically sufficient.

## Operational Notes

- DNS-01 wildcard requires access to DNS provider API
- Treat the DNS API token as a secret — store in `forge secrets`, not in `.env` files committed to Git
- Consider certificate rotation procedures if multiple admins exist
