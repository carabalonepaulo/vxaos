-- Initial schema for VXA-OS Go server (SQLite).
-- Character aggregate collections (inventory, equips, quests, switches, ...)
-- are stored as versioned JSON in character_state; core queryable stats and
-- position stay in characters.

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    username_key TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    password_algo TEXT NOT NULL,
    password_salt BLOB NOT NULL,
    password_iterations INTEGER NOT NULL,
    password_verifier BLOB NOT NULL,
    group_id INTEGER NOT NULL DEFAULT 0,
    vip_until INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    last_login_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
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
    gold INTEGER NOT NULL DEFAULT 0,
    guild_id INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    last_login_at INTEGER NOT NULL DEFAULT 0,
    UNIQUE (account_id, slot)
);

CREATE TABLE character_state (
    character_id INTEGER PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    data TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE bank_state (
    account_id INTEGER PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    gold INTEGER NOT NULL DEFAULT 0,
    data TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE guilds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL UNIQUE,
    leader_character_id INTEGER NOT NULL DEFAULT 0,
    notice TEXT NOT NULL DEFAULT '',
    flag TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE TABLE character_friends (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    friend_character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    PRIMARY KEY (character_id, friend_character_id)
);

CREATE TABLE bans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    value TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    expires_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    created_by INTEGER NOT NULL DEFAULT 0,
    UNIQUE (kind, value)
);

CREATE TABLE server_state (
    state_key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    at INTEGER NOT NULL,
    account_id INTEGER NOT NULL DEFAULT 0,
    action TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_characters_account ON characters(account_id);
CREATE INDEX idx_characters_guild ON characters(guild_id);
CREATE INDEX idx_character_friends_friend ON character_friends(friend_character_id);
CREATE INDEX idx_bans_expires ON bans(expires_at);
CREATE INDEX idx_audit_at ON audit_log(at);
