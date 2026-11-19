-- Initial schema for VXA-OS Go server (PostgreSQL 14+).
-- Character aggregate collections live in character_state.data as versioned
-- JSON; queryable core stats and position stay in characters.

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    username_key TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    password_algo TEXT NOT NULL,
    password_salt BYTEA NOT NULL,
    password_iterations INTEGER NOT NULL,
    password_verifier BYTEA NOT NULL,
    group_id INTEGER NOT NULL DEFAULT 0,
    vip_until BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    last_login_at BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE characters (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    slot INTEGER NOT NULL,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL UNIQUE,
    class_id INTEGER NOT NULL,
    sex INTEGER NOT NULL,
    level INTEGER NOT NULL,
    exp INTEGER NOT NULL,
    hp INTEGER NOT NULL,
    mp INTEGER NOT NULL,
    points INTEGER NOT NULL,
    mhp INTEGER NOT NULL,
    mmp INTEGER NOT NULL,
    atk INTEGER NOT NULL,
    defense INTEGER NOT NULL,
    mat INTEGER NOT NULL,
    mdef INTEGER NOT NULL,
    agi INTEGER NOT NULL,
    luk INTEGER NOT NULL,
    character_name TEXT NOT NULL,
    character_index INTEGER NOT NULL,
    face_name TEXT NOT NULL,
    face_index INTEGER NOT NULL,
    map_id INTEGER NOT NULL,
    x INTEGER NOT NULL,
    y INTEGER NOT NULL,
    direction INTEGER NOT NULL,
    revive_map_id INTEGER NOT NULL,
    revive_x INTEGER NOT NULL,
    revive_y INTEGER NOT NULL,
    gold BIGINT NOT NULL DEFAULT 0,
    guild_id BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    last_login_at BIGINT NOT NULL DEFAULT 0,
    UNIQUE (account_id, slot)
);

CREATE TABLE character_state (
    character_id BIGINT PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    data TEXT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE bank_state (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    gold BIGINT NOT NULL DEFAULT 0,
    data TEXT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE guilds (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL UNIQUE,
    leader_character_id BIGINT NOT NULL DEFAULT 0,
    notice TEXT NOT NULL DEFAULT '',
    flag TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL
);

CREATE TABLE character_friends (
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    friend_character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    PRIMARY KEY (character_id, friend_character_id)
);

CREATE TABLE bans (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    value TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    expires_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    created_by BIGINT NOT NULL DEFAULT 0,
    UNIQUE (kind, value)
);

CREATE TABLE server_state (
    state_key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    at BIGINT NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0,
    action TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_characters_account ON characters(account_id);
CREATE INDEX idx_characters_guild ON characters(guild_id);
CREATE INDEX idx_character_friends_friend ON character_friends(friend_character_id);
CREATE INDEX idx_bans_expires ON bans(expires_at);
CREATE INDEX idx_audit_at ON audit_log(at);
