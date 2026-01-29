-- Audit logs table for tracking important actions
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    category VARCHAR(30) NOT NULL,
    details JSONB DEFAULT '{}',
    ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_category ON audit_logs(category);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Composite index for filtering by user and category
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_category ON audit_logs(user_id, category);

-- Partial index for game logs (most common)
CREATE INDEX IF NOT EXISTS idx_audit_logs_games ON audit_logs(user_id, created_at DESC) WHERE category = 'game';
