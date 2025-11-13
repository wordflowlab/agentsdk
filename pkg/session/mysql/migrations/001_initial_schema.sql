-- AgentSDK Session MySQL Schema
-- Version: 1.0
-- Date: 2025-11-13
-- Description: Initial schema for session management (MySQL 8.0+)
-- Requirements: MySQL 8.0+ (for JSON support)

-- ============================================================
-- Table: sessions
-- Description: Main session table
-- ============================================================
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_user_sessions (user_id, updated_at DESC),
    INDEX idx_app_sessions (app_name, updated_at DESC),
    INDEX idx_agent_sessions (agent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Table: session_states
-- Description: Hierarchical state storage (app/user/session/temp)
-- ============================================================
CREATE TABLE IF NOT EXISTS session_states (
    session_id VARCHAR(36) NOT NULL,
    scope VARCHAR(50) NOT NULL,
    `key` VARCHAR(255) NOT NULL,  -- 使用反引号转义保留字
    value JSON NOT NULL,            -- MySQL 8.0+ JSON 类型
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (session_id, scope, `key`),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,

    INDEX idx_state_scope (session_id, scope)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Table: session_events
-- Description: Event log for each session
-- ============================================================
CREATE TABLE IF NOT EXISTS session_events (
    id VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    invocation_id VARCHAR(255) NOT NULL,
    branch VARCHAR(500) NOT NULL,
    author VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Content stored as JSON
    content JSON NOT NULL,

    -- Actions stored as JSON
    actions JSON,

    -- Long running tool IDs (JSON array)
    long_running_tool_ids JSON,

    -- Metadata
    metadata JSON,

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,

    INDEX idx_session_events (session_id, timestamp DESC),
    INDEX idx_invocation_events (invocation_id),
    INDEX idx_branch_events (session_id, branch(255))  -- 限制索引长度
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Table: session_artifacts
-- Description: Artifact version management
-- ============================================================
CREATE TABLE IF NOT EXISTS session_artifacts (
    id VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    content LONGBLOB,
    mime_type VARCHAR(100),
    size BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,

    INDEX idx_artifacts (session_id, name, version DESC),
    UNIQUE INDEX idx_artifact_version (session_id, name, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Stored Procedures
-- ============================================================

-- Get session statistics
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS get_session_stats(IN p_session_id VARCHAR(36))
BEGIN
    SELECT
        (SELECT COUNT(*) FROM session_events WHERE session_id = p_session_id) AS event_count,
        (SELECT COUNT(*) FROM session_states WHERE session_id = p_session_id) AS state_count,
        (SELECT COUNT(*) FROM session_artifacts WHERE session_id = p_session_id) AS artifact_count,
        (SELECT MIN(timestamp) FROM session_events WHERE session_id = p_session_id) AS first_event_time,
        (SELECT MAX(timestamp) FROM session_events WHERE session_id = p_session_id) AS last_event_time;
END //
DELIMITER ;

-- Clean up old sessions
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS cleanup_old_sessions(IN days_to_keep INT)
BEGIN
    DELETE FROM sessions
    WHERE updated_at < DATE_SUB(NOW(), INTERVAL days_to_keep DAY);

    SELECT ROW_COUNT() AS deleted_count;
END //
DELIMITER ;

-- ============================================================
-- JSON Virtual Columns (Optional, for better query performance)
-- ============================================================

-- Uncomment to add virtual columns for frequently queried JSON fields

/*
ALTER TABLE session_events
ADD COLUMN event_role VARCHAR(50) AS (JSON_UNQUOTE(JSON_EXTRACT(content, '$.role'))) VIRTUAL,
ADD INDEX idx_event_role (event_role);

ALTER TABLE session_events
ADD COLUMN has_tool_calls BOOLEAN AS (JSON_CONTAINS_PATH(content, 'one', '$.tool_calls')) VIRTUAL,
ADD INDEX idx_has_tool_calls (has_tool_calls);
*/

-- ============================================================
-- Sample Data (for development/testing)
-- ============================================================

-- Uncomment the following lines to insert sample data

/*
INSERT INTO sessions (id, app_name, user_id, agent_id)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'demo-app', 'user-1', 'agent-1'),
    ('00000000-0000-0000-0000-000000000002', 'demo-app', 'user-2', 'agent-1');

INSERT INTO session_states (session_id, scope, `key`, value)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'app', 'version', JSON_OBJECT('value', '1.0.0')),
    ('00000000-0000-0000-0000-000000000001', 'user', 'theme', JSON_QUOTE('dark')),
    ('00000000-0000-0000-0000-000000000001', 'session', 'page', CAST(1 AS JSON));
*/

-- ============================================================
-- Verification Queries
-- ============================================================

-- List all tables
-- SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE();

-- Check table structure
-- DESCRIBE sessions;
-- DESCRIBE session_states;
-- DESCRIBE session_events;
-- DESCRIBE session_artifacts;

-- Check indexes
-- SHOW INDEX FROM sessions;
-- SHOW INDEX FROM session_events;

-- Check foreign keys
-- SELECT
--     TABLE_NAME,
--     COLUMN_NAME,
--     CONSTRAINT_NAME,
--     REFERENCED_TABLE_NAME,
--     REFERENCED_COLUMN_NAME
-- FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
-- WHERE REFERENCED_TABLE_NAME IS NOT NULL
--     AND TABLE_SCHEMA = DATABASE();

-- ============================================================
-- Performance Tips
-- ============================================================

/*
1. JSON 列优化:
   - 使用虚拟列为常用 JSON 字段创建索引
   - 避免在 WHERE 子句中直接查询 JSON 深层字段

2. 分区表（大数据量场景）:
   ALTER TABLE session_events
   PARTITION BY RANGE (UNIX_TIMESTAMP(timestamp)) (
       PARTITION p202501 VALUES LESS THAN (UNIX_TIMESTAMP('2025-02-01')),
       PARTITION p202502 VALUES LESS THAN (UNIX_TIMESTAMP('2025-03-01')),
       ...
   );

3. 定期清理:
   -- 每周执行
   CALL cleanup_old_sessions(90);

4. 监控查询性能:
   EXPLAIN SELECT * FROM session_events WHERE session_id = '...';
*/
