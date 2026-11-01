# Backend guidelines

- Stack: Go + Fiber + Bun ORM + Postgres, with JWT-based auth.
- Keep handlers thin; put domain logic in services and persistence in database/virtualfs/blob layers.
- Keep packages focused: auth, user, account, catalog (HTTP for VFS), upload, virtualfs, blob, database.
- Use public IDs in API responses and inputs; internal UUIDs stay server-side.

# Services and stores

- Expose services as interfaces with private implementations and a compile-time assertion (`var _ Service = (*service)(nil)`).
- Inject service interfaces into handlers. Keep request parsing, response mapping, cookies, and streaming in the HTTP layer.
- Inject stores into user, account, organization, and drive services; keep their SQL in `db_store.go`.
- Application services own transactions. Construct participating stores with the same `bun.Tx`; pass it to VFS and auth operations that join the transaction.
- Registration, login, refresh, share mutations, and catalog mutations return results only after a successful commit.
- Refresh-token rejection can require committing revocation or cleanup. Preserve that behavior for both endpoints and cookie middleware.
- Lock refresh-token rows during rotation and verify the secret before reuse handling. Allow a fixed 10-second grace from the original `ConsumedAt`, without issuing tokens or extending grace. Auto-refresh collisions keep valid access authentication and leave cookies untouched; refresh-only collisions return 409 and require the latest token. Verified reuse after grace revokes the grant.
- Resolve organization membership and drive/share access in services so handlers and middleware use the same rules.

# Routing + auth conventions

- Drive-scoped resources live under `/drives/:driveID`; always apply auth + drive middleware.
- Auth middleware must be the source of truth for the current user (via `reqctx`).
- Support both bearer-token and cookie flows; pick one per client surface.
- Use transactions for multi-step writes or cross-table changes.

# Data model relationships (high level)

- Users have accounts (principals) within organizations.
- Drives own VFS nodes (files + directories).
- Auth grants own refresh tokens.
- Node shares grant scoped access into drives.

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
- Keep CORS and cookie settings aligned with the frontend deployment topology; cookie `Secure` is derived from request protocol.
