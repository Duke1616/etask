package dao

import (
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/pkg/sqlx"
)

// AIConversation 是 AI 会话持久化对象。
type AIConversation struct {
	ID        int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`
	TenantID  int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index:idx_ai_conversation_project,priority:1;comment:'租户ID'"`
	UserID    int64  `gorm:"column:user_id;type:bigint unsigned;not null;index;comment:'创建用户ID'"`
	ProjectID int64  `gorm:"column:project_id;type:bigint;not null;index:idx_ai_conversation_project,priority:2;comment:'Codebook项目ID'"`
	Title     string `gorm:"column:title;type:varchar(128);not null;comment:'会话标题'"`
	Provider  string `gorm:"column:provider;type:varchar(32);not null;comment:'模型供应商'"`
	Model     string `gorm:"column:model;type:varchar(128);not null;comment:'模型名称'"`
	Status    string `gorm:"column:status;type:varchar(16);not null;index;comment:'会话状态'"`
	RunToken  string `gorm:"column:run_token;type:varchar(64);not null;default:'';comment:'当前生成租约令牌'"`
	CTime     int64  `gorm:"column:ctime;type:bigint;not null;comment:'创建时间'"`
	UTime     int64  `gorm:"column:utime;type:bigint;not null;comment:'更新时间'"`
}

func (AIConversation) TableName() string { return "ai_conversation" }

// AIMessage 是 AI 会话消息持久化对象。
type AIMessage struct {
	ID             int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`
	TenantID       int64  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index:idx_ai_message_conversation,priority:1;comment:'租户ID'"`
	ConversationID int64  `gorm:"column:conversation_id;type:bigint;not null;index:idx_ai_message_conversation,priority:2;comment:'AI会话ID'"`
	Role           string `gorm:"column:role;type:varchar(16);not null;comment:'消息角色'"`
	Content        string `gorm:"column:content;type:longtext;not null;comment:'消息内容'"`
	Status         string `gorm:"column:status;type:varchar(16);not null;index;comment:'生成状态'"`
	Provider       string `gorm:"column:provider;type:varchar(32);not null;default:'';comment:'模型供应商'"`
	Model          string `gorm:"column:model;type:varchar(128);not null;default:'';comment:'模型名称'"`
	RecipeID       string `gorm:"column:recipe_id;type:varchar(64);not null;default:'';comment:'场景标识'"`
	RecipeVersion  string `gorm:"column:recipe_version;type:varchar(32);not null;default:'';comment:'场景版本'"`
	InputTokens    int64  `gorm:"column:input_tokens;type:bigint;not null;default:0;comment:'输入Token数'"`
	OutputTokens   int64  `gorm:"column:output_tokens;type:bigint;not null;default:0;comment:'输出Token数'"`
	LatencyMillis  int64  `gorm:"column:latency_millis;type:bigint;not null;default:0;comment:'响应耗时毫秒'"`
	ErrorMessage   string `gorm:"column:error_message;type:text;not null;comment:'失败原因'"`
	CTime          int64  `gorm:"column:ctime;type:bigint;not null;comment:'创建时间'"`
	UTime          int64  `gorm:"column:utime;type:bigint;not null;comment:'更新时间'"`
}

func (AIMessage) TableName() string { return "ai_message" }

// AISuggestion 是 AI 候选代码持久化对象。
type AISuggestion struct {
	ID               int64                                  `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`
	TenantID         int64                                  `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;index:idx_ai_suggestion_conversation,priority:1;comment:'租户ID'"`
	ConversationID   int64                                  `gorm:"column:conversation_id;type:bigint;not null;index:idx_ai_suggestion_conversation,priority:2;comment:'AI会话ID'"`
	MessageID        int64                                  `gorm:"column:message_id;type:bigint;not null;index;comment:'模型消息ID'"`
	ProjectID        int64                                  `gorm:"column:project_id;type:bigint;not null;index;comment:'Codebook项目ID'"`
	NodeID           int64                                  `gorm:"column:node_id;type:bigint;not null;index;comment:'Codebook节点ID'"`
	BaseVersionID    int64                                  `gorm:"column:base_version_id;type:bigint;not null;comment:'生成时基础版本ID'"`
	BaseHash         string                                 `gorm:"column:base_hash;type:char(64);not null;comment:'基础版本摘要'"`
	RecipeID         string                                 `gorm:"column:recipe_id;type:varchar(64);not null;comment:'场景标识'"`
	RecipeVersion    string                                 `gorm:"column:recipe_version;type:varchar(32);not null;comment:'场景版本'"`
	Language         string                                 `gorm:"column:language;type:varchar(16);not null;comment:'脚本语言'"`
	Code             string                                 `gorm:"column:code;type:longtext;not null;comment:'候选代码'"`
	Summary          string                                 `gorm:"column:summary;type:text;not null;comment:'修改摘要'"`
	Diagnostics      sqlx.JSONColumn[[]domain.AIDiagnostic] `gorm:"column:diagnostics;type:json;comment:'静态诊断结果'"`
	Status           string                                 `gorm:"column:status;type:varchar(16);not null;index;comment:'候选状态'"`
	AppliedVersionID int64                                  `gorm:"column:applied_version_id;type:bigint;not null;default:0;comment:'应用后版本ID'"`
	CTime            int64                                  `gorm:"column:ctime;type:bigint;not null;comment:'创建时间'"`
	UTime            int64                                  `gorm:"column:utime;type:bigint;not null;comment:'更新时间'"`
}

func (AISuggestion) TableName() string { return "ai_suggestion" }
