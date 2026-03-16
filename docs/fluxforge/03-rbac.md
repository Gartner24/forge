# Roles and RBAC

FluxForge implements three roles. Role escalation is not permitted.

## Roles

| Role | Who | Capabilities |
|---|---|---|
| **Owner** | First admin (set during `forge fluxforge init`) | All operations including promoting admins and reassigning the controller |
| **Admin** | Nodes promoted by the Owner | Generate tokens, revoke nodes, view mesh status |
| **Node** | All other joined nodes | Participate in the mesh. No management capabilities. |

## Role Assignment

```bash
# Promote a node to Admin (Owner only)
forge fluxforge add-admin <node-id>

# Remove Admin role (Owner only)
forge fluxforge remove-admin <node-id>
```

## Controller Reassignment

The Owner can designate a different node as the FluxController:

```bash
forge fluxforge set-controller <node-id>
```

The new controller must already be a mesh member. During the transition, the old controller stops accepting new connections while the new one takes over the peer registry. Existing WireGuard tunnels remain up throughout.

## Auth on Every Request

The FluxController refreshes its peer list and role assignments on every incoming request — there is no cached auth state. Revoking a node or removing an Admin role takes effect on the next request from that node without restarting the controller.
