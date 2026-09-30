-- 000041: editable role permissions (Settings → Roles & permissions).
-- A row replaces the built-in permission set of that role; no row means the
-- built-in defaults. SUPER_ADMIN is never overridden.
CREATE TABLE IF NOT EXISTS role_permission_overrides (
    role        TEXT PRIMARY KEY CHECK (role IN ('ADMIN', 'MANAGER', 'DISPATCH', 'ACCOUNTANT', 'CLEANER')),
    permissions TEXT[] NOT NULL DEFAULT '{}',
    updated_by  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
