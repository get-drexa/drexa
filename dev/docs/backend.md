# Backend guidelines

- Stack: Go + Fiber + Bun ORM + Postgres, with JWT-based auth.
- Keep handlers thin; put domain logic in services and persistence in database/virtualfs/blob layers.
- Keep packages focused: auth, user, account, catalog (HTTP for VFS), upload, virtualfs, blob, database.
- Use public IDs in API responses and inputs; internal UUIDs stay server-side.

# Routing + auth conventions

- Account-scoped resources live under `/accounts/:accountID`; always apply auth + account middleware.
- Auth middleware must be the source of truth for the current user (via `reqctx`).
- Support both bearer-token and cookie flows; pick one per client surface.
- Use transactions for multi-step writes or cross-table changes.

# Data model relationships (high level)

- Users own accounts.
- Accounts own VFS nodes (files + directories).
- Auth grants own refresh tokens.
- Node share tokens exist for future sharing flows.

# Virtual filesystem + storage

- Files and directories are both VFS nodes; use the VirtualFS API for create/move/rename/delete.
- Soft delete supports “trash” behavior; permanent delete removes data + blobs.
- Storage mode/backends influence blob keys; do not assume a single strategy.
- Any operation that changes paths must keep the blob store in sync.

# Uploads

- Uploads are staged then finalized; treat pending uploads as ephemeral.
- Finalization is responsible for setting metadata and moving the node to a usable state.

# Error handling

- Wrap failures with `httperr` helpers; return JSON `{ "error": "..." }`.
- Prefer explicit status codes for user-visible failures; log internal errors.

# Docs + tooling

- Keep swag annotations current when endpoints or response shapes change.
- Update the OpenAPI patching logic when you introduce new union-like types.

# Configuration

- Config is YAML-driven; see `apps/backend/config.example.yaml` for required fields and env overrides.
- Keep CORS and cookie settings aligned with the frontend deployment topology.
