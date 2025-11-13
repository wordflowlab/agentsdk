package mysql

import (
	"time"
)

// SessionModel MySQL 会话模型
// 对应表: sessions
type SessionModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	AppName   string    `gorm:"type:varchar(255);not null;index:idx_app_sessions"`
	UserID    string    `gorm:"type:varchar(255);not null;index:idx_user_sessions"`
	AgentID   string    `gorm:"type:varchar(255);not null;index:idx_agent_sessions"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;index:idx_user_sessions,idx_app_sessions"`

	// 关联关系
	States    []StateModel    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Events    []EventModel    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Artifacts []ArtifactModel `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (SessionModel) TableName() string {
	return "sessions"
}

// StateModel MySQL 状态模型
// 对应表: session_states
type StateModel struct {
	SessionID string    `gorm:"primaryKey;type:varchar(36)"`
	Scope     string    `gorm:"primaryKey;type:varchar(50)"`
	Key       string    `gorm:"primaryKey;type:varchar(255);column:key"` // key 是 MySQL 保留字，需要转义
	Value     []byte    `gorm:"type:json;not null"`                       // MySQL 8.0+ JSON 类型
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`

	// 关联关系
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (StateModel) TableName() string {
	return "session_states"
}

// EventModel MySQL 事件模型
// 对应表: session_events
type EventModel struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"`
	SessionID    string    `gorm:"type:varchar(36);not null;index:idx_session_events"`
	InvocationID string    `gorm:"type:varchar(255);not null;index:idx_invocation_events"`
	Branch       string    `gorm:"type:varchar(500);not null;index:idx_branch_events"`
	Author       string    `gorm:"type:varchar(255);not null"`
	AgentID      string    `gorm:"type:varchar(255);not null"`
	Timestamp    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_session_events"`

	// 内容 - JSON 存储
	Content []byte `gorm:"type:json;not null"`

	// 动作 - JSON 存储
	Actions []byte `gorm:"type:json"`

	// 长时运行工具 ID 列表 - JSON 数组存储
	LongRunningToolIDs []byte `gorm:"type:json"`

	// 元数据 - JSON 存储
	Metadata []byte `gorm:"type:json"`

	// 关联关系
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (EventModel) TableName() string {
	return "session_events"
}

// ArtifactModel MySQL 工件模型
// 对应表: session_artifacts
type ArtifactModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	SessionID string    `gorm:"type:varchar(36);not null;index:idx_artifacts"`
	Name      string    `gorm:"type:varchar(255);not null;index:idx_artifacts"`
	Version   int       `gorm:"not null;index:idx_artifacts"`
	Content   []byte    `gorm:"type:longblob"`
	MimeType  string    `gorm:"type:varchar(100)"`
	Size      int64     `gorm:"default:0"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 唯一约束：同一 session 下，同一工件的同一版本只能有一条记录
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (ArtifactModel) TableName() string {
	return "session_artifacts"
}

// BeforeCreate GORM 钩子
func (ArtifactModel) BeforeCreate() error {
	return nil
}

// MySQL 索引定义（通过 GORM AutoMigrate 自动创建）
// CREATE INDEX idx_user_sessions ON sessions(user_id, updated_at DESC);
// CREATE INDEX idx_app_sessions ON sessions(app_name, updated_at DESC);
// CREATE INDEX idx_agent_sessions ON sessions(agent_id);
// CREATE INDEX idx_session_events ON session_events(session_id, timestamp DESC);
// CREATE INDEX idx_invocation_events ON session_events(invocation_id);
// CREATE INDEX idx_branch_events ON session_events(session_id, branch);
// CREATE INDEX idx_artifacts ON session_artifacts(session_id, name, version DESC);
// CREATE UNIQUE INDEX idx_artifact_version ON session_artifacts(session_id, name, version);

// MySQL 8.0+ JSON 列注意事项:
// 1. JSON 列支持虚拟列索引:
//    CREATE INDEX idx_content_role ON session_events((CAST(content->>'$.role' AS CHAR(50))));
//
// 2. JSON 搜索:
//    SELECT * FROM session_events WHERE JSON_CONTAINS(content, '"user"', '$.role');
//
// 3. JSON 路径查询:
//    SELECT JSON_EXTRACT(content, '$.content') FROM session_events;
