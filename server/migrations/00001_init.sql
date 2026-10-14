-- Core schema. See docs/requirements.md sections 3-10.
-- Secret material is stored only as client-side ciphertext (bytea); the server
-- never sees plaintext item data, names of folders, or unwrapped keys (SEC-02).

-- +goose Up

-- Users are instance-level accounts (section 3). Keys are generated on the client.
CREATE TABLE users (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 text        NOT NULL,
    name                  text        NOT NULL DEFAULT '',
    -- Server-side hash of the client-derived authentication hash (AUTH-01).
    auth_hash             text        NOT NULL,
    kdf_algorithm         text        NOT NULL DEFAULT 'argon2id',
    kdf_params            jsonb       NOT NULL,
    kdf_salt              bytea       NOT NULL,
    public_key            bytea       NOT NULL,
    encrypted_private_key bytea       NOT NULL,
    encrypted_user_key    bytea       NOT NULL,
    totp_secret_encrypted bytea,
    is_instance_admin     boolean     NOT NULL DEFAULT false,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    disabled_at           timestamptz
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL UNIQUE,
    client_type  text        NOT NULL CHECK (client_type IN ('web', 'extension', 'cli')),
    device_name  text        NOT NULL DEFAULT '',
    ip           inet,
    user_agent   text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

-- Organizations are tenants (section 4). Every tenant-owned row carries org_id (ORG-04).
CREATE TABLE organizations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    slug       text        NOT NULL UNIQUE,
    settings   jsonb       NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    suspended_at timestamptz,
    deleted_at timestamptz
);

-- Roles hold administrative permissions only; seeing secrets needs a grant (section 6).
CREATE TABLE roles (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name          text        NOT NULL,
    system_key    text CHECK (system_key IN ('owner', 'admin', 'manager', 'member', 'auditor')),
    permissions   text[]      NOT NULL DEFAULT '{}',
    -- For manager-type roles: folder and group ids the role may administer.
    managed_scope jsonb       NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, name)
);
CREATE UNIQUE INDEX roles_system_key ON roles (org_id, system_key) WHERE system_key IS NOT NULL;

CREATE TYPE member_status AS ENUM ('invited', 'accepted', 'confirmed', 'suspended');

CREATE TABLE members (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             uuid          NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id            uuid REFERENCES users (id) ON DELETE CASCADE,
    email              text          NOT NULL,
    role_id            uuid          NOT NULL REFERENCES roles (id),
    status             member_status NOT NULL DEFAULT 'invited',
    -- Organization key wrapped to the member's public key, set at confirmation (USR-05).
    encrypted_org_key  bytea,
    invited_by         uuid REFERENCES members (id) ON DELETE SET NULL,
    invite_token_hash  bytea UNIQUE,
    invite_expires_at  timestamptz,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    confirmed_at       timestamptz,
    suspended_at       timestamptz
);
CREATE UNIQUE INDEX members_org_user_key ON members (org_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX members_org_email_key ON members (org_id, lower(email));

CREATE TABLE groups (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (org_id, name)
);

CREATE TABLE group_members (
    group_id  uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    member_id uuid NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    org_id    uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, member_id)
);

-- Folders (section 7). kind = 'project' marks an env-var project (section 12).
CREATE TABLE folders (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    parent_id      uuid REFERENCES folders (id) ON DELETE RESTRICT,
    kind           text        NOT NULL DEFAULT 'folder' CHECK (kind IN ('folder', 'project', 'personal')),
    -- Personal folders belong to one member and are invisible to admins (USR-13).
    owner_member_id uuid REFERENCES members (id) ON DELETE CASCADE,
    -- Name, description, colour and icon, encrypted with the organization key.
    encrypted_meta bytea       NOT NULL,
    -- Empty array = any item type allowed (FLD-02).
    allowed_types  text[]      NOT NULL DEFAULT '{}',
    key_version    integer     NOT NULL DEFAULT 1,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);
CREATE INDEX folders_org_parent_idx ON folders (org_id, parent_id);

-- The folder key wrapped to each member who can open the folder (section 9, step 5).
CREATE TABLE folder_keys (
    folder_id     uuid    NOT NULL REFERENCES folders (id) ON DELETE CASCADE,
    member_id     uuid    NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    org_id        uuid    NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    key_version   integer NOT NULL,
    encrypted_key bytea   NOT NULL,
    PRIMARY KEY (folder_id, member_id, key_version)
);

-- Items (section 7). Type is visible to the server for folder type rules; all fields are ciphertext.
CREATE TABLE items (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    folder_id          uuid        NOT NULL REFERENCES folders (id) ON DELETE RESTRICT,
    type               text        NOT NULL,
    -- Item key wrapped by the folder key at folder_key_version.
    wrapped_item_key   bytea       NOT NULL,
    folder_key_version integer     NOT NULL,
    encrypted_data     bytea       NOT NULL,
    revision           integer     NOT NULL DEFAULT 1,
    created_by         uuid REFERENCES members (id) ON DELETE SET NULL,
    updated_by         uuid REFERENCES members (id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);
CREATE INDEX items_org_folder_idx ON items (org_id, folder_id) WHERE deleted_at IS NULL;

-- Version history (ITM-06). Keeps the last 20 revisions per item.
CREATE TABLE item_versions (
    item_id        uuid        NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    revision       integer     NOT NULL,
    org_id         uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    encrypted_data bytea       NOT NULL,
    created_by     uuid REFERENCES members (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, revision)
);

-- Item key wrapped directly to a member, for selected-item grants (section 9, step 6).
CREATE TABLE item_keys (
    item_id       uuid  NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    member_id     uuid  NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    org_id        uuid  NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    encrypted_key bytea NOT NULL,
    PRIMARY KEY (item_id, member_id)
);

-- Grants (section 8): subject x level x scope, optionally expiring.
CREATE TYPE access_level AS ENUM ('use', 'view', 'edit', 'manage');

CREATE TABLE grants (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             uuid         NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    subject_type       text         NOT NULL CHECK (subject_type IN ('member', 'group')),
    subject_id         uuid         NOT NULL,
    scope_type         text         NOT NULL CHECK (scope_type IN ('org', 'folder', 'item')),
    scope_id           uuid,
    include_subfolders boolean      NOT NULL DEFAULT true,
    level              access_level NOT NULL,
    expires_at         timestamptz,
    created_by         uuid REFERENCES members (id) ON DELETE SET NULL,
    created_at         timestamptz  NOT NULL DEFAULT now(),
    CHECK ((scope_type = 'org') = (scope_id IS NULL))
);
CREATE INDEX grants_subject_idx ON grants (org_id, subject_type, subject_id);
CREATE INDEX grants_scope_idx ON grants (org_id, scope_type, scope_id);

-- Append-only, hash-chained audit log (AUD-01, AUD-02).
CREATE TABLE audit_events (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- No foreign key: events outlive a deleted organization until retention expires (AUD-04).
    org_id      uuid,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_type  text        NOT NULL CHECK (actor_type IN ('member', 'user', 'token', 'system')),
    actor_id    uuid,
    actor_email text        NOT NULL DEFAULT '',
    action      text        NOT NULL,
    target_type text        NOT NULL DEFAULT '',
    target_id   uuid,
    result      text        NOT NULL DEFAULT 'success' CHECK (result IN ('success', 'denied', 'error')),
    ip          inet,
    user_agent  text        NOT NULL DEFAULT '',
    device_id   text        NOT NULL DEFAULT '',
    metadata    jsonb       NOT NULL DEFAULT '{}',
    prev_hash   bytea,
    hash        bytea       NOT NULL
);
CREATE INDEX audit_events_org_time_idx ON audit_events (org_id, occurred_at DESC);

-- +goose StatementBegin
CREATE FUNCTION audit_events_block_change() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER audit_events_append_only
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_block_change();

-- +goose Down
DROP TABLE audit_events;
DROP FUNCTION audit_events_block_change;
DROP TABLE grants;
DROP TYPE access_level;
DROP TABLE item_keys;
DROP TABLE item_versions;
DROP TABLE items;
DROP TABLE folder_keys;
DROP TABLE folders;
DROP TABLE group_members;
DROP TABLE groups;
DROP TABLE members;
DROP TYPE member_status;
DROP TABLE roles;
DROP TABLE organizations;
DROP TABLE sessions;
DROP TABLE users;
