CREATE TABLE IF NOT EXISTS families (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- SQLite не умеет ADD COLUMN с NOT NULL + FK в старых версиях,
-- поэтому добавляем как NULL-able и заполняем позже по мере надобности.
ALTER TABLE users ADD COLUMN family_id TEXT;

ALTER TABLE users ADD COLUMN role TEXT;

-- ALTER TABLE tasks ADD COLUMN family_id TEXT;

CREATE INDEX IF NOT EXISTS idx_users_family_id ON users (family_id);

CREATE INDEX IF NOT EXISTS idx_tasks_family_id ON tasks (family_id);

CREATE INDEX IF NOT EXISTS idx_families_code ON families (invite_code);