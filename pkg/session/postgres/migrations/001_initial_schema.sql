-- AgentSDK Session PostgreSQL Schema
-- Version: 1.0
-- Date: 2025-11-13
-- Description: Initial schema for session management

-- ============================================================
-- Extension: Enable UUID generation
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- Table: sessions
-- Description: Main session table
-- ============================================================
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for sessions
CREATE INDEX IF NOT EXISTS idx_user_sessions ON sessions(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_sessions ON sessions(app_name, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_sessions ON sessions(agent_id, updated_at DESC);

-- Comment
COMMENT ON TABLE sessions IS 'User-Agent interaction sessions';
COMMENT ON COLUMN sessions.id IS 'Session unique identifier';
COMMENT ON COLUMN sessions.app_name IS 'Application name';
COMMENT ON COLUMN sessions.user_id IS 'User identifier';
COMMENT ON COLUMN sessions.agent_id IS 'Agent identifier';

-- ============================================================
-- Table: session_states
-- Description: Hierarchical state storage (app/user/session/temp)
-- ============================================================
CREATE TABLE IF NOT EXISTS session_states (
    session_id UUID NOT NULL,
    scope VARCHAR(50) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (session_id, scope, key),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Indexes for session_states
CREATE INDEX IF NOT EXISTS idx_state_scope ON session_states(session_id, scope);
CREATE INDEX IF NOT EXISTS idx_state_search ON session_states USING gin(value);

-- Comment
COMMENT ON TABLE session_states IS 'Hierarchical state storage';
COMMENT ON COLUMN session_states.scope IS 'State scope: app, user, session, temp';
COMMENT ON COLUMN session_states.value IS 'JSONB stored value';

-- ============================================================
-- Table: session_events
-- Description: Event log for each session
-- ============================================================
CREATE TABLE IF NOT EXISTS session_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    invocation_id VARCHAR(255) NOT NULL,
    branch VARCHAR(500) NOT NULL,
    author VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Content stored as JSONB
    content JSONB NOT NULL,

    -- Actions stored as JSONB
    actions JSONB,

    -- Long running tool IDs
    long_running_tool_ids TEXT[],

    -- Metadata
    metadata JSONB,

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Indexes for session_events
CREATE INDEX IF NOT EXISTS idx_session_events ON session_events(session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_invocation_events ON session_events(invocation_id);
CREATE INDEX IF NOT EXISTS idx_branch_events ON session_events(session_id, branch);
CREATE INDEX IF NOT EXISTS idx_events_content_search ON session_events USING gin(content);
CREATE INDEX IF NOT EXISTS idx_events_actions_search ON session_events USING gin(actions);

-- Comment
COMMENT ON TABLE session_events IS 'Event log for agent-user interactions';
COMMENT ON COLUMN session_events.invocation_id IS 'Unique invocation identifier for tracing';
COMMENT ON COLUMN session_events.branch IS 'Agent branch path (e.g., root.search.analyze)';
COMMENT ON COLUMN session_events.author IS 'Event author (user/agent/system)';
COMMENT ON COLUMN session_events.content IS 'Event content (message, tool calls, etc.)';
COMMENT ON COLUMN session_events.actions IS 'Event actions (state delta, transfer, etc.)';

-- ============================================================
-- Table: session_artifacts
-- Description: Artifact version management
-- ============================================================
CREATE TABLE IF NOT EXISTS session_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    content BYTEA,
    mime_type VARCHAR(100),
    size BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Indexes for session_artifacts
CREATE INDEX IF NOT EXISTS idx_artifacts ON session_artifacts(session_id, name, version DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_artifact_version ON session_artifacts(session_id, name, version);

-- Comment
COMMENT ON TABLE session_artifacts IS 'Artifact version tracking';
COMMENT ON COLUMN session_artifacts.name IS 'Artifact name (e.g., report.pdf)';
COMMENT ON COLUMN session_artifacts.version IS 'Artifact version number';
COMMENT ON COLUMN session_artifacts.content IS 'Artifact binary content';

-- ============================================================
-- Function: Update updated_at timestamp
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for sessions table
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for session_states table
CREATE TRIGGER update_states_updated_at
    BEFORE UPDATE ON session_states
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Function: Get session statistics
-- ============================================================
CREATE OR REPLACE FUNCTION get_session_stats(p_session_id UUID)
RETURNS TABLE (
    event_count BIGINT,
    state_count BIGINT,
    artifact_count BIGINT,
    first_event_time TIMESTAMPTZ,
    last_event_time TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT COUNT(*) FROM session_events WHERE session_id = p_session_id),
        (SELECT COUNT(*) FROM session_states WHERE session_id = p_session_id),
        (SELECT COUNT(*) FROM session_artifacts WHERE session_id = p_session_id),
        (SELECT MIN(timestamp) FROM session_events WHERE session_id = p_session_id),
        (SELECT MAX(timestamp) FROM session_events WHERE session_id = p_session_id);
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Function: Clean up old sessions
-- ============================================================
CREATE OR REPLACE FUNCTION cleanup_old_sessions(days_to_keep INT DEFAULT 90)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM sessions
    WHERE updated_at < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Comment
COMMENT ON FUNCTION get_session_stats IS 'Get statistics for a session';
COMMENT ON FUNCTION cleanup_old_sessions IS 'Delete sessions older than specified days';

-- ============================================================
-- Sample Data (for development/testing)
-- ============================================================
-- Uncomment the following lines to insert sample data

/*
INSERT INTO sessions (id, app_name, user_id, agent_id)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'demo-app', 'user-1', 'agent-1'),
    ('00000000-0000-0000-0000-000000000002', 'demo-app', 'user-2', 'agent-1');

INSERT INTO session_states (session_id, scope, key, value)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'app', 'version', '"1.0.0"'::jsonb),
    ('00000000-0000-0000-0000-000000000001', 'user', 'theme', '"dark"'::jsonb),
    ('00000000-0000-0000-0000-000000000001', 'session', 'page', '1'::jsonb);
*/

-- ============================================================
-- Verification Queries
-- ============================================================
-- Run these queries to verify the schema

-- List all tables
-- SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';

-- List all indexes
-- SELECT indexname, tablename FROM pg_indexes WHERE schemaname = 'public';

-- Check foreign keys
-- SELECT conname, conrelid::regclass, confrelid::regclass
-- FROM pg_constraint WHERE contype = 'f';
