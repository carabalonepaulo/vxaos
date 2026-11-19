-- Initial schema for VXA-OS Go server (MySQL 8+).
-- Character aggregate collections live in character_state.data as versioned
-- JSON; queryable core stats and position stay in characters.

CREATE TABLE accounts (
    id BIGINT NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    username_key VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL DEFAULT '',
    password_algo VARCHAR(32) NOT NULL,
    password_salt VARBINARY(64) NOT NULL,
    password_iterations INT NOT NULL,
    password_verifier VARBINARY(128) NOT NULL,
    group_id INT NOT NULL DEFAULT 0,
    vip_until BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    last_login_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY ux_accounts_username_key (username_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE characters (
    id BIGINT NOT NULL AUTO_INCREMENT,
    account_id BIGINT NOT NULL,
    slot INT NOT NULL,
    name VARCHAR(64) NOT NULL,
    name_key VARCHAR(64) NOT NULL,
    class_id INT NOT NULL,
    sex INT NOT NULL,
    level INT NOT NULL,
    exp INT NOT NULL,
    hp INT NOT NULL,
    mp INT NOT NULL,
    points INT NOT NULL,
    mhp INT NOT NULL,
    mmp INT NOT NULL,
    atk INT NOT NULL,
    defense INT NOT NULL,
    mat INT NOT NULL,
    mdef INT NOT NULL,
    agi INT NOT NULL,
    luk INT NOT NULL,
    character_name VARCHAR(128) NOT NULL,
    character_index INT NOT NULL,
    face_name VARCHAR(128) NOT NULL,
    face_index INT NOT NULL,
    map_id INT NOT NULL,
    x INT NOT NULL,
    y INT NOT NULL,
    direction INT NOT NULL,
    revive_map_id INT NOT NULL,
    revive_x INT NOT NULL,
    revive_y INT NOT NULL,
    gold BIGINT NOT NULL DEFAULT 0,
    guild_id BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    last_login_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY ux_characters_name_key (name_key),
    UNIQUE KEY ux_characters_account_slot (account_id, slot),
    CONSTRAINT fk_characters_account FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE character_state (
    character_id BIGINT NOT NULL,
    version INT NOT NULL,
    data LONGTEXT NOT NULL,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (character_id),
    CONSTRAINT fk_character_state_character FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE bank_state (
    account_id BIGINT NOT NULL,
    version INT NOT NULL,
    gold BIGINT NOT NULL DEFAULT 0,
    data LONGTEXT NOT NULL,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (account_id),
    CONSTRAINT fk_bank_state_account FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE guilds (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    name_key VARCHAR(64) NOT NULL,
    leader_character_id BIGINT NOT NULL DEFAULT 0,
    notice VARCHAR(512) NOT NULL DEFAULT '',
    flag VARCHAR(512) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY ux_guilds_name_key (name_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE character_friends (
    character_id BIGINT NOT NULL,
    friend_character_id BIGINT NOT NULL,
    PRIMARY KEY (character_id, friend_character_id),
    CONSTRAINT fk_friends_character FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    CONSTRAINT fk_friends_friend FOREIGN KEY (friend_character_id) REFERENCES characters(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE bans (
    id BIGINT NOT NULL AUTO_INCREMENT,
    kind VARCHAR(16) NOT NULL,
    value VARCHAR(255) NOT NULL,
    reason VARCHAR(255) NOT NULL DEFAULT '',
    expires_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    created_by BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY ux_bans_kind_value (kind, value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE server_state (
    state_key VARCHAR(64) NOT NULL,
    value LONGTEXT NOT NULL,
    PRIMARY KEY (state_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE audit_log (
    id BIGINT NOT NULL AUTO_INCREMENT,
    at BIGINT NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0,
    action VARCHAR(64) NOT NULL,
    details TEXT NOT NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_characters_account ON characters(account_id);
CREATE INDEX idx_characters_guild ON characters(guild_id);
CREATE INDEX idx_character_friends_friend ON character_friends(friend_character_id);
CREATE INDEX idx_bans_expires ON bans(expires_at);
CREATE INDEX idx_audit_at ON audit_log(at);
