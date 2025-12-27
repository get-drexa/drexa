-- ============================================================================
-- Application Tables
-- ============================================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    display_name TEXT,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_usage_bytes BIGINT NOT NULL DEFAULT 0,
    storage_quota_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS grants (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY,
    grant_id UUID NOT NULL REFERENCES grants(id) ON DELETE CASCADE,
    key UUID NOT NULL UNIQUE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_grant_id ON refresh_tokens(grant_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Virtual filesystem nodes (unified files + directories)
CREATE TABLE IF NOT EXISTS vfs_nodes (
    id UUID PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,  -- opaque ID for external API (no timestamp leak)
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES vfs_nodes(id) ON DELETE CASCADE,  -- NULL = root directory
    kind TEXT NOT NULL CHECK (kind IN ('file', 'directory')),
    status TEXT NOT NULL DEFAULT 'ready' CHECK (status IN ('pending', 'ready')),
    name TEXT NOT NULL,
    
    -- File-specific fields (NULL for directories)
    blob_key TEXT,      -- reference to blob storage (flat mode), NULL for hierarchical
    size BIGINT,        -- file size in bytes
    mime_type TEXT,     -- content type
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,  -- soft delete for trash
    
    -- No duplicate names in same parent (per account, excluding deleted)
    CONSTRAINT unique_node_name UNIQUE NULLS NOT DISTINCT (account_id, parent_id, name, deleted_at)
);

CREATE INDEX idx_vfs_nodes_account_id ON vfs_nodes(account_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vfs_nodes_parent_id ON vfs_nodes(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vfs_nodes_account_parent ON vfs_nodes(account_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vfs_nodes_kind ON vfs_nodes(account_id, kind) WHERE deleted_at IS NULL;
CREATE INDEX idx_vfs_nodes_deleted ON vfs_nodes(account_id, deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_vfs_nodes_public_id ON vfs_nodes(public_id);
CREATE UNIQUE INDEX idx_vfs_nodes_account_root ON vfs_nodes(account_id) WHERE parent_id IS NULL;  -- one root per account
CREATE INDEX idx_vfs_nodes_pending ON vfs_nodes(created_at) WHERE status = 'pending';  -- for cleanup job

CREATE TABLE IF NOT EXISTS node_shares (
    id UUID PRIMARY KEY,
    -- the account that owns the share
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    public_id TEXT NOT NULL UNIQUE,  -- opaque ID for external API (no timestamp leak)
    -- parent directory of the items in this share
    shared_directory_id UUID NOT NULL REFERENCES vfs_nodes(id) ON DELETE CASCADE,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_node_shares_public_id ON node_shares(public_id);
CREATE INDEX idx_node_shares_shared_directory_id ON node_shares(shared_directory_id);
CREATE INDEX idx_node_shares_expires_at ON node_shares(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS share_permissions (
    id UUID PRIMARY KEY,
    share_id UUID NOT NULL REFERENCES node_shares(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,  -- NULL = anyone with the link
    can_read BOOLEAN NOT NULL DEFAULT TRUE,
    can_write BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete BOOLEAN NOT NULL DEFAULT FALSE,
    can_upload BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (share_id, account_id)
);

CREATE INDEX idx_share_permissions_share_id ON share_permissions(share_id);
CREATE INDEX idx_share_permissions_account_id ON share_permissions(account_id) WHERE account_id IS NOT NULL;
CREATE INDEX idx_share_permissions_expires_at ON share_permissions(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS share_items (
    id UUID PRIMARY KEY,
    share_id UUID NOT NULL REFERENCES node_shares(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES vfs_nodes(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (share_id, node_id)
);

CREATE INDEX idx_share_items_share_id ON share_items(share_id);
CREATE INDEX idx_share_items_node_id ON share_items(node_id);

-- ============================================================================
-- Triggers for updated_at timestamps
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_vfs_nodes_updated_at BEFORE UPDATE ON vfs_nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_node_shares_updated_at BEFORE UPDATE ON node_shares
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_share_permissions_updated_at BEFORE UPDATE ON share_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_share_items_updated_at BEFORE UPDATE ON share_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_grants_updated_at BEFORE UPDATE ON grants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
