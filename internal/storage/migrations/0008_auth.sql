-- Email и пароль. Существующие пользователи остаются, но без входа.
ALTER TABLE users ADD COLUMN email TEXT;

ALTER TABLE users ADD COLUMN password_hash TEXT;

-- Уникальность email только среди тех, у кого он задан.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email)
WHERE
    email IS NOT NULL;

-- Имена больше не уникальны: два «Папы» в разных семьях — норма.
DROP INDEX IF EXISTS idx_users_name_ci;

-- Инвайты
CREATE TABLE IF NOT EXISTS invites (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    family_id TEXT NOT NULL REFERENCES families (id) ON DELETE CASCADE,
    created_by TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_by TEXT,
    used_at TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invites_family_id ON invites (family_id);

CREATE INDEX IF NOT EXISTS idx_invites_code ON invites (code);

-- Сессии
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);

CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions (expires_at);