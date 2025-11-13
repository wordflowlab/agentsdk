-- AgentSDK Session PostgreSQL Schema Rollback
-- Version: 1.0
-- Date: 2025-11-13
-- Description: Rollback initial schema for session management

-- ============================================================
-- Drop Functions
-- ============================================================
DROP FUNCTION IF EXISTS cleanup_old_sessions(INT);
DROP FUNCTION IF EXISTS get_session_stats(UUID);
DROP FUNCTION IF EXISTS update_updated_at_column();

-- ============================================================
-- Drop Tables (in reverse order due to foreign keys)
-- ============================================================
DROP TABLE IF EXISTS session_artifacts CASCADE;
DROP TABLE IF EXISTS session_events CASCADE;
DROP TABLE IF EXISTS session_states CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;

-- ============================================================
-- Drop Extensions (optional, comment out if used by other schemas)
-- ============================================================
-- DROP EXTENSION IF EXISTS "uuid-ossp";
