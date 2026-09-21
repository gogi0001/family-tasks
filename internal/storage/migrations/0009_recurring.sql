CREATE TABLE IF NOT EXISTS task_templates (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL REFERENCES families (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    assignee TEXT NOT NULL,
    created_by TEXT NOT NULL,
    rule TEXT NOT NULL,
    next_run_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

ALTER TABLE tasks ADD COLUMN template_id TEXT;

ALTER TABLE tasks ADD COLUMN scheduled_for TEXT;

CREATE INDEX IF NOT EXISTS idx_templates_family ON task_templates (family_id);

CREATE INDEX IF NOT EXISTS idx_templates_next_run ON task_templates (active, next_run_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_template_scheduled ON tasks (template_id, scheduled_for)
WHERE
    template_id IS NOT NULL;