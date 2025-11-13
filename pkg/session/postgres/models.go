package postgres

import (
	"time"

	"github.com/lib/pq"
)

// SessionModel 会话数据库模型
// 对应表: sessions
type SessionModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AppName   string    `gorm:"type:varchar(255);not null;index:idx_app_sessions"`
	UserID    string    `gorm:"type:varchar(255);not null;index:idx_user_sessions"`
	AgentID   string    `gorm:"type:varchar(255);not null;index:idx_agent_sessions"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now();index:idx_user_sessions,idx_app_sessions"`

	// 关联关系
	States    []StateModel    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Events    []EventModel    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Artifacts []ArtifactModel `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (SessionModel) TableName() string {
	return "sessions"
}

// StateModel 状态数据库模型
// 对应表: session_states
// 支持分层状态: app/user/session/temp
type StateModel struct {
	SessionID string      `gorm:"primaryKey;type:uuid"`
	Scope     string      `gorm:"primaryKey;type:varchar(50)"` // app, user, session, temp
	Key       string      `gorm:"primaryKey;type:varchar(255)"`
	Value     []byte      `gorm:"type:jsonb;not null"` // JSONB 存储
	CreatedAt time.Time   `gorm:"not null;default:now()"`
	UpdatedAt time.Time   `gorm:"not null;default:now()"`

	// 关联关系
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (StateModel) TableName() string {
	return "session_states"
}

// EventModel 事件数据库模型
// 对应表: session_events
type EventModel struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID    string         `gorm:"type:uuid;not null;index:idx_session_events"`
	InvocationID string         `gorm:"type:varchar(255);not null;index:idx_invocation_events"`
	Branch       string         `gorm:"type:varchar(500);not null;index:idx_branch_events"`
	Author       string         `gorm:"type:varchar(255);not null"`
	AgentID      string         `gorm:"type:varchar(255);not null"`
	Timestamp    time.Time      `gorm:"not null;default:now();index:idx_session_events"`

	// 内容 - JSONB 存储
	Content  []byte `gorm:"type:jsonb;not null"`

	// 动作 - JSONB 存储
	Actions  []byte `gorm:"type:jsonb"`

	// 长时运行工具 ID 列表
	LongRunningToolIDs pq.StringArray `gorm:"type:text[]"`

	// 元数据 - JSONB 存储
	Metadata []byte `gorm:"type:jsonb"`

	// 关联关系
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (EventModel) TableName() string {
	return "session_events"
}

// ArtifactModel 工件版本数据库模型
// 对应表: session_artifacts
// 支持 EventActions.ArtifactDelta
type ArtifactModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID string    `gorm:"type:uuid;not null;index:idx_artifacts"`
	Name      string    `gorm:"type:varchar(255);not null;index:idx_artifacts"`
	Version   int       `gorm:"not null;index:idx_artifacts"`
	Content   []byte    `gorm:"type:bytea"`
	MimeType  string    `gorm:"type:varchar(100)"`
	Size      int64     `gorm:"default:0"`
	CreatedAt time.Time `gorm:"not null;default:now()"`

	// 唯一约束: 同一 session 下，同一工件的同一版本只能有一条记录
	// 通过 GORM 索引实现
	Session SessionModel `gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (ArtifactModel) TableName() string {
	return "session_artifacts"
}

// 添加复合唯一索引
func (ArtifactModel) BeforeCreate() error {
	// GORM 会自动处理唯一索引
	return nil
}

// 索引定义（通过 GORM AutoMigrate 自动创建）
// CREATE INDEX idx_user_sessions ON sessions(user_id, updated_at DESC);
// CREATE INDEX idx_app_sessions ON sessions(app_name, updated_at DESC);
// CREATE INDEX idx_agent_sessions ON sessions(agent_id, updated_at DESC);
// CREATE INDEX idx_state_scope ON session_states(session_id, scope);
// CREATE INDEX idx_session_events ON session_events(session_id, timestamp DESC);
// CREATE INDEX idx_invocation_events ON session_events(invocation_id);
// CREATE INDEX idx_branch_events ON session_events(session_id, branch);
// CREATE INDEX idx_artifacts ON session_artifacts(session_id, name, version DESC);
// CREATE UNIQUE INDEX idx_artifact_version ON session_artifacts(session_id, name, version);
