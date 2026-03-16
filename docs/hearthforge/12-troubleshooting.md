# Troubleshooting

This document provides a checklist for common HearthForge issues.

## VS Code / Cursor / JetBrains Cannot Connect

**Checklist:**
- Does the target hostname resolve to the VPS IP?
- Is the gateway reachable on port 2224?
- Is the dev container running?
- Is `sshd` running inside the dev container?
- Is the developer key present and mapped correctly?
- Is the container reachable from the gateway (network `dev-web`)?

**Remote-SSH specific:**
- Use the final host alias (`<dev>-<project>`) that includes `ProxyJump`
- Do not test with a direct interactive gateway shell
- Ensure `IdentityFile` points to the right private key
- Add `IdentitiesOnly yes` to avoid wrong-key attempts

**Useful checks:**
```bash
docker ps | grep dev-
docker exec -it <dev-container> ps aux | grep sshd
docker network inspect dev-web
```

---

## SSH Works But No Shell / Permission Issues

- Confirm the container user exists and has a home directory
- Confirm `sshd_config` allows the intended user
- Confirm workspace permissions under `/workspace/<project>`

**If you see:** `channel open failed: administratively prohibited: Rejected`
→ This is expected when trying to open a shell directly on the gateway.
→ Fix: connect using the final alias with `ProxyJump` (`ssh <dev>-<project>`).

**If you see:** `Permission denied (publickey)` on gateway hop
```bash
# Confirm key in gateway canonical store
ls /opt/infra/forge/hearthforge/gateway/authorized_keys/<dev>.pub

# Validate key format
sudo ssh-keygen -lf /opt/infra/forge/hearthforge/gateway/authorized_keys/<dev>.pub

# Re-register key
forge hearthforge gateway-add-key --dev <dev> --pubkey "<full ssh-ed25519 ... line>"
```

---

## Proxy Routes to Wrong Target

- Verify vhost file exists in `proxy/conf.d/active/`
- Validate proxy config:
  ```bash
  docker exec -it nginx-proxy nginx -t
  ```
- Check for duplicate `server_name` collisions

---

## ACME / Cert Renewal Issues

- Confirm port 80 is open on the VPS
- Confirm `/.well-known/acme-challenge/` is routed to the webroot directory
- Confirm certbot container is running and has correct volumes

If SmeltForge/Caddy is used, cert renewal is automatic — check Caddy logs instead:
```bash
docker logs caddy --tail=50
```

---

## Container DNS Name Not Resolving

- Confirm proxy and target container share the same network
- For dev routing, confirm both are on `dev-web`
- For prod routing, confirm both are on `web`

---

## Dev Container Cannot Access Internet / Git

- Confirm container has outbound connectivity
- Confirm DNS resolution works inside container:
  ```bash
  docker exec -it dev-<project>-<dev> nslookup github.com
  ```

---

## Disk Usage Issues

- Check `/opt/data/dev_workspaces/` size:
  ```bash
  du -sh /opt/data/dev_workspaces/*/
  ```
- Prune unused containers/images (admin only):
  ```bash
  docker container prune
  docker image prune
  ```
- Implement workspace retention policy — see [Offboarding](10-offboarding.md)

---

## Quick Verification Commands

```bash
# Proxy config test
docker exec -it nginx-proxy nginx -t

# Proxy networks
docker inspect nginx-proxy | grep -A6 Networks

# Dev container networks
docker inspect dev-<project>-<dev> | grep -A6 Networks

# Dev container sshd
docker exec -it dev-<project>-<dev> ss -lntp | grep :22

# Gateway status
docker ps | grep gateway
sudo ss -lntp | grep 2224

# Recent gateway events
sudo docker logs proxy-gateway-1 --tail=50

# Recent auth attempts
sudo rg "result=accepted|result=rejected" \
  /opt/infra/forge/hearthforge/gateway/logs/audit.log | tail -20
```
