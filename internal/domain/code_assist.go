package domain

import (
	"fmt"
	"strings"

	"github.com/Duke1616/etask/internal/errs"
)

// AIConversationStatus 表示 AI 会话的运行状态。
type AIConversationStatus string

const (
	AIConversationStatusActive  AIConversationStatus = "ACTIVE"
	AIConversationStatusRunning AIConversationStatus = "RUNNING"
)

// AIMessageRole 表示对话消息角色。
type AIMessageRole string

const (
	AIMessageRoleUser      AIMessageRole = "USER"
	AIMessageRoleAssistant AIMessageRole = "ASSISTANT"
)

// AIMessageStatus 表示消息生成状态。
type AIMessageStatus string

const (
	AIMessageStatusStreaming AIMessageStatus = "STREAMING"
	AIMessageStatusCompleted AIMessageStatus = "COMPLETED"
	AIMessageStatusFailed    AIMessageStatus = "FAILED"
	AIMessageStatusCancelled AIMessageStatus = "CANCELLED"
)

// AISuggestionStatus 表示候选代码状态。
type AISuggestionStatus string

const (
	AISuggestionStatusDraft     AISuggestionStatus = "DRAFT"
	AISuggestionStatusValidated AISuggestionStatus = "VALIDATED"
	AISuggestionStatusApplying  AISuggestionStatus = "APPLYING"
	AISuggestionStatusApplied   AISuggestionStatus = "APPLIED"
)

// AIDiagnosticSeverity 表示候选代码诊断级别。
type AIDiagnosticSeverity string

const (
	AIDiagnosticSeverityError   AIDiagnosticSeverity = "ERROR"
	AIDiagnosticSeverityWarning AIDiagnosticSeverity = "WARNING"
)

// AIConversation 是一个项目内的持久化 AI 对话。
type AIConversation struct {
	ID        int64
	TenantID  int64
	UserID    int64
	ProjectID int64
	Title     string
	Provider  string
	Model     string
	Status    AIConversationStatus
	CTime     int64
	UTime     int64
}

// ValidateCreate 校验创建会话所需的信息。
func (c *AIConversation) ValidateCreate() error {
	c.Title = strings.TrimSpace(c.Title)
	if c.UserID <= 0 || c.ProjectID <= 0 {
		return fmt.Errorf("%w: invalid AI conversation owner or project", errs.ErrInvalidParameter)
	}
	if c.Title == "" {
		c.Title = "新对话"
	}
	if len(c.Title) > 128 {
		return fmt.Errorf("%w: AI conversation title is too long", errs.ErrInvalidParameter)
	}
	return nil
}

// AIMessage 是会话中的一条用户或模型消息。
type AIMessage struct {
	ID             int64
	TenantID       int64
	ConversationID int64
	Role           AIMessageRole
	Content        string
	Status         AIMessageStatus
	Provider       string
	Model          string
	RecipeID       string
	RecipeVersion  string
	InputTokens    int64
	OutputTokens   int64
	LatencyMillis  int64
	ErrorMessage   string
	CTime          int64
	UTime          int64
}

// AIDiagnostic 描述候选代码的确定性检查结果。
type AIDiagnostic struct {
	Severity AIDiagnosticSeverity `json:"severity"`
	Code     string               `json:"code"`
	Message  string               `json:"message"`
}

// AISuggestion 是模型针对一个 Codebook 文件生成的候选代码。
type AISuggestion struct {
	ID               int64
	TenantID         int64
	ConversationID   int64
	MessageID        int64
	ProjectID        int64
	NodeID           int64
	BaseVersionID    int64
	BaseHash         string
	RecipeID         string
	RecipeVersion    string
	Language         string
	Code             string
	Summary          string
	Diagnostics      []AIDiagnostic
	Status           AISuggestionStatus
	AppliedVersionID int64
	CTime            int64
	UTime            int64
}

// Prepare 规范化并校验候选代码。
func (s *AISuggestion) Prepare() error {
	s.Language = strings.ToLower(strings.TrimSpace(s.Language))
	s.Summary = strings.TrimSpace(s.Summary)
	if s.ConversationID <= 0 || s.MessageID <= 0 || s.ProjectID <= 0 ||
		s.NodeID <= 0 || s.BaseVersionID <= 0 {
		return fmt.Errorf("invalid AI suggestion context")
	}
	if strings.TrimSpace(s.Code) == "" {
		return fmt.Errorf("AI suggestion code is empty")
	}
	if strings.TrimSpace(s.RecipeID) == "" || strings.TrimSpace(s.RecipeVersion) == "" {
		return fmt.Errorf("AI suggestion recipe is missing")
	}
	if s.Language != "python" && s.Language != "shell" {
		return fmt.Errorf("unsupported AI suggestion language: %s", s.Language)
	}
	if s.Status == "" {
		s.Status = AISuggestionStatusDraft
	}
	return nil
}

// AIChatContext 描述一次对话附带的编辑器上下文。
type AIChatContext struct {
	NodeID        int64
	BaseVersionID int64
	EditorCode    string
}

// AIChatRequest 描述用户发送的一条 AI 消息。
type AIChatRequest struct {
	ConversationID int64
	RecipeID       string
	Content        string
	Context        AIChatContext
}
