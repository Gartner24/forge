# Joining the Mesh and Token Management

## Initialising the First Node

```bash
forge fluxforge init
```

Run this once on the server that will be the controller. This:
1. Generates the controller WireGuard keypair
2. Starts FluxController on port 7777
3. Assigns this node mesh IP `10.forge.1.1`
4. Creates the Owner role for the current admin
5. Generates the first join token and prints it

Custom port:
```bash
forge fluxforge init --port 8777
```

## Generating Join Tokens

```bash
forge fluxforge token create
```

Tokens are:
- Cryptographically random (32+ bytes)
- Single-use — consumed on first successful join
- Valid for 24 hours

List pending tokens:
```bash
forge fluxforge token list
```

Revoke before use:
```bash
forge fluxforge token revoke <token>
```

## Joining a New Node

Run on the server being added to the mesh:

```bash
forge fluxforge join \
  --controller <controller-public-ip>:7777 \
  --token <token>
```

What happens:
1. System validates the token with the controller
2. Generates a WireGuard keypair on the new node
3. Sends the public key to the controller
4. Controller assigns a mesh IP (e.g. `10.forge.2.1`)
5. Controller pushes the new peer to all existing agents
6. WireGuard is configured on all nodes
7. System confirms with the new node's mesh IP

If direct WireGuard connection between peers fails due to NAT, the controller relays traffic until a direct path is negotiated.

## Revoking a Node

```bash
forge fluxforge revoke <node-id>
```

Within 5 seconds:
- Node is removed from the peer registry
- Updated peer list is pushed to all remaining agents
- WireGuard tunnel to the revoked node is torn down on all agents
- The revoked node cannot rejoin without a new token

## Promoting a Node to Admin Role

```bash
forge fluxforge add-admin <node-id>
```

Only the Owner can promote nodes. See [Roles and RBAC](03-rbac.md).
