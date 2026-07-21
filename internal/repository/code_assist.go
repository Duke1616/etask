package repository

import (
	"context"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
)

// CodeAssistRepository 定义 AI 会话和候选代码仓储能力。
type CodeAssistRepository interface {
	// CreateConversation 创建 AI 会话。
	CreateConversation(ctx context.Context, conversation domain.AIConversation) (domain.AIConversation, error)
	// GetConversationByID 查询 AI 会话。
	GetConversationByID(ctx context.Context, id int64) (domain.AIConversation, error)
	// ListConversations 查询用户在项目下的 AI 会话。
	ListConversations(ctx context.Context, userID, projectID int64, limit int) ([]domain.AIConversation, error)
	// ClaimConversation 原子占用 AI 会话。
	ClaimConversation(ctx context.Context, id, userID int64, runToken string) error
	// ReleaseConversation 释放 AI 会话。
	ReleaseConversation(ctx context.Context, id int64, runToken string) error
	// CreateMessage 创建 AI 消息。
	CreateMessage(ctx context.Context, message domain.AIMessage) (domain.AIMessage, error)
	// CompleteMessage 保存模型回复和用量。
	CompleteMessage(ctx context.Context, message domain.AIMessage) error
	// FailMessage 标记模型消息失败。
	FailMessage(ctx context.Context, message domain.AIMessage,
		status domain.AIMessageStatus, errorMessage string) error
	// ListMessages 查询 AI 会话消息。
	ListMessages(ctx context.Context, conversationID int64, limit int) ([]domain.AIMessage, error)
	// CreateSuggestion 创建候选代码。
	CreateSuggestion(ctx context.Context, suggestion domain.AISuggestion) (domain.AISuggestion, error)
	// GetSuggestionByID 查询候选代码。
	GetSuggestionByID(ctx context.Context, id int64) (domain.AISuggestion, error)
	// ListSuggestions 查询会话中的候选代码。
	ListSuggestions(ctx context.Context, conversationID int64) ([]domain.AISuggestion, error)
	// ClaimSuggestion 原子占用待应用候选代码。
	ClaimSuggestion(ctx context.Context, id int64) error
	// ReleaseSuggestion 释放应用失败的候选代码。
	ReleaseSuggestion(ctx context.Context, id int64, status domain.AISuggestionStatus) error
	// MarkSuggestionApplied 原子记录候选代码应用结果。
	MarkSuggestionApplied(ctx context.Context, id, versionID int64) error
}

type codeAssistRepository struct{ dao *dao.GORMCodeAssistDAO }

// NewCodeAssistRepository 创建 AI 仓储。
func NewCodeAssistRepository(source *dao.GORMCodeAssistDAO) CodeAssistRepository {
	return &codeAssistRepository{dao: source}
}

func (r *codeAssistRepository) CreateConversation(ctx context.Context,
	conversation domain.AIConversation) (domain.AIConversation, error) {
	created, err := r.dao.CreateConversation(ctx, toAIConversationEntity(conversation))
	return toAIConversationDomain(created), err
}

func (r *codeAssistRepository) GetConversationByID(ctx context.Context,
	id int64) (domain.AIConversation, error) {
	conversation, err := r.dao.GetConversationByID(ctx, id)
	return toAIConversationDomain(conversation), err
}

func (r *codeAssistRepository) ListConversations(ctx context.Context, userID, projectID int64,
	limit int) ([]domain.AIConversation, error) {
	entities, err := r.dao.ListConversations(ctx, userID, projectID, limit)
	result := make([]domain.AIConversation, 0, len(entities))
	for _, entity := range entities {
		result = append(result, toAIConversationDomain(entity))
	}
	return result, err
}

func (r *codeAssistRepository) ClaimConversation(ctx context.Context, id, userID int64,
	runToken string) error {
	return r.dao.ClaimConversation(ctx, id, userID, runToken)
}

func (r *codeAssistRepository) ReleaseConversation(ctx context.Context, id int64,
	runToken string) error {
	return r.dao.ReleaseConversation(ctx, id, runToken)
}

func (r *codeAssistRepository) CreateMessage(ctx context.Context,
	message domain.AIMessage) (domain.AIMessage, error) {
	created, err := r.dao.CreateMessage(ctx, toAIMessageEntity(message))
	return toAIMessageDomain(created), err
}

func (r *codeAssistRepository) CompleteMessage(ctx context.Context, message domain.AIMessage) error {
	return r.dao.CompleteMessage(ctx, toAIMessageEntity(message))
}

func (r *codeAssistRepository) FailMessage(ctx context.Context, message domain.AIMessage,
	status domain.AIMessageStatus, errorMessage string) error {
	return r.dao.FailMessage(ctx, toAIMessageEntity(message), string(status), errorMessage)
}

func (r *codeAssistRepository) ListMessages(ctx context.Context, conversationID int64,
	limit int) ([]domain.AIMessage, error) {
	entities, err := r.dao.ListMessages(ctx, conversationID, limit)
	result := make([]domain.AIMessage, 0, len(entities))
	for _, entity := range entities {
		result = append(result, toAIMessageDomain(entity))
	}
	return result, err
}

func (r *codeAssistRepository) CreateSuggestion(ctx context.Context,
	suggestion domain.AISuggestion) (domain.AISuggestion, error) {
	created, err := r.dao.CreateSuggestion(ctx, toAISuggestionEntity(suggestion))
	return toAISuggestionDomain(created), err
}

func (r *codeAssistRepository) GetSuggestionByID(ctx context.Context,
	id int64) (domain.AISuggestion, error) {
	suggestion, err := r.dao.GetSuggestionByID(ctx, id)
	return toAISuggestionDomain(suggestion), err
}

func (r *codeAssistRepository) ListSuggestions(ctx context.Context,
	conversationID int64) ([]domain.AISuggestion, error) {
	entities, err := r.dao.ListSuggestions(ctx, conversationID)
	result := make([]domain.AISuggestion, 0, len(entities))
	for _, entity := range entities {
		result = append(result, toAISuggestionDomain(entity))
	}
	return result, err
}

func (r *codeAssistRepository) ClaimSuggestion(ctx context.Context, id int64) error {
	return r.dao.ClaimSuggestion(ctx, id)
}

func (r *codeAssistRepository) ReleaseSuggestion(ctx context.Context, id int64,
	status domain.AISuggestionStatus) error {
	return r.dao.ReleaseSuggestion(ctx, id, string(status))
}

func (r *codeAssistRepository) MarkSuggestionApplied(ctx context.Context, id, versionID int64) error {
	return r.dao.MarkSuggestionApplied(ctx, id, versionID)
}
