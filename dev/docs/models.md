# Data models (users, orgs, accounts, drives)

This doc describes the current model for identity, org membership, and storage tenancy.

## Core ideas

- `Account` is a **principal** (a user’s identity within an org), not a storage tenant.
- `Drive` is the **storage tenant** (the partition key for VFS + blobs).
- VFS authorization separates:
  - **tenant**: which drive is being accessed, and
  - **actor**: who is performing the action.

## Entities

### User

Authenticated human identity.

### Organization

Top-level container for org-owned resources:

- billing, policies, API keys, integrations, audit log, etc (all keyed by `org_id`)

### Account (principal)

Represents a user’s identity within an org (membership).

Fields:

- `org_id`, `user_id`
- `role` (org-level role), `status` (invited/active/removed/etc)
- timestamps

Invariant:

- `UNIQUE(org_id, user_id)` (one account per user per org)

### Drive (storage tenant)

Storage container that owns the filesystem.

Fields:

- `org_id`
- optional `owner_account_id` (for “personal drive inside this org”)
- quota/usage metadata (quotas are per-drive)
- timestamps

Invariants:

- Personal drive: `UNIQUE(org_id, owner_account_id)` (one personal drive per account per org).
- Shared/org drive: `owner_account_id IS NULL`.

## Relationships

```
users 1 ── * accounts * ── 1 organizations
organizations 1 ── * drives 1 ── * vfs_nodes
```

Interpretation:

- A user can be in many orgs (via many accounts).
- An org can have many members (many accounts).
- Storage is owned by drives (org-scoped), not by accounts.

## VFS tenancy + scope

`Scope` should carry the VFS tenant key and actor identity separately:

- `Scope.DriveID` = the storage tenant partition key (used to constrain all VFS reads/writes).
- `Scope.RootNodeID` + `Scope.AllowedNodes` = the accessible boundary within that drive.
- `Scope.ActorKind` + `Scope.ActorID` = who performs the action (authenticated account vs share link).

## Sharing model

Separate “where the shared content lives” from “who created/manages the share record”.

Shape:

```
node_shares.drive_id                = storage tenant containing the shared nodes
node_shares.created_by_account_id   = principal that created the share

share_permissions.account_id        = principal allowed (NULL = public)
```

Consumption semantics:

- `Scope.DriveID` comes from `node_shares.drive_id` (tenant).
- `Scope.Actor…` comes from either the share link (public) or the consuming principal (authenticated consumption).

## Drive patterns

Two common patterns can coexist:

- **Personal drives**: one private drive per account per org (`owner_account_id` set). This matches “each user gets their own drive in every org they join”.
- **Org-owned/shared drives**: `owner_account_id = NULL`, access controlled via org role/policies (and optionally per-drive ACLs).

“Personal orgs” are just orgs with a single member account and a single personal drive.

## Authorization checks (high level)

- Org-owned resources: user can access org `X` iff they have an active `accounts` row `(org_id=X, user_id=U)`.
- Drive access: user can access drive `D` iff they’re a member of `D.org_id` and their role/policies allow the requested operation.
