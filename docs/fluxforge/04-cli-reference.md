# CLI Reference — FluxForge

## forge fluxforge init

```bash
forge fluxforge init
forge fluxforge init --port 8777
```

Initialises the mesh controller on this node. Run once on the designated controller server.

---

## forge fluxforge join

```bash
forge fluxforge join --controller <ip>:<port> --token <token>
```

Joins this server to an existing mesh. Run on the server being added.

---

## forge fluxforge status

```bash
forge fluxforge status
forge fluxforge status --output json
```

Shows controller status, mesh subnet, and a summary of connected nodes.

---

## forge fluxforge nodes

```bash
forge fluxforge nodes
forge fluxforge nodes --output json
```

Lists all registered nodes with ID, mesh IP, role, and last-seen timestamp.

---

## forge fluxforge token

```bash
forge fluxforge token create           # generate a 24h single-use token
forge fluxforge token list             # list pending tokens
forge fluxforge token revoke <token>   # revoke before use
```

---

## forge fluxforge revoke

```bash
forge fluxforge revoke <node-id>
```

Removes a node from the mesh immediately. All peers stop routing to it within 5 seconds.

---

## forge fluxforge add-admin / remove-admin

```bash
forge fluxforge add-admin <node-id>
forge fluxforge remove-admin <node-id>
```

Owner only.

---

## forge fluxforge set-controller

```bash
forge fluxforge set-controller <node-id>
```

Reassigns the FluxController to a different mesh node. Owner only.

---

## forge fluxforge ping

```bash
forge fluxforge ping <node-id>
forge fluxforge ping <mesh-ip>
```

Tests reachability to a specific mesh node over the WireGuard interface.
