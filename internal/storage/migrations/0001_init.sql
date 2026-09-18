CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    family_id TEXT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    assignee TEXT NOT NULL,
    created_by TEXT NOT NULL,
    status TEXT NOT NULL CHECK (
        status IN ('todo', 'in_progress', 'done')
    ),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks (created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_family_id ON tasks (family_id);