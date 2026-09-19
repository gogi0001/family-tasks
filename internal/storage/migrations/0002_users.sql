CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Уникальность имени без учёта регистра (для find-or-create).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_name_ci ON users (lower(name));