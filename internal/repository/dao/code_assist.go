package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"gorm.io/gorm"
)

const (
	aiConversationClaimTimeout = 10 * time.Minute
	aiSuggestionClaimTimeout   = time.Minute
)

// GORMCodeAssistDAO 使用 GORM 持久化 AI 数据。
type GORMCodeAssistDAO struct{ db *gorm.DB }

// NewGORMCodeAssistDAO 创建 AI 持久化对象。
func NewGORMCodeAssistDAO(db *gorm.DB) *GORMCodeAssistDAO { return &GORMCodeAssistDAO{db: db} }

func (g *GORMCodeAssistDAO) CreateConversation(ctx context.Context,
	conversation AIConversation) (AIConversation, error) {
	now := time.Now().UnixMilli()
	conversation.CTime, conversation.UTime = now, now
	err := g.db.WithContext(ctx).Create(&conversation).Error
	return conversation, err
}

func (g *GORMCodeAssistDAO) GetConversationByID(ctx context.Context, id int64) (AIConversation, error) {
	var conversation AIConversation
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&conversation).Error
	return conversation, err
}

func (g *GORMCodeAssistDAO) ListConversations(ctx context.Context, userID, projectID int64,
	limit int) ([]AIConversation, error) {
	var conversations []AIConversation
	err := g.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Order("utime DESC").Limit(limit).Find(&conversations).Error
	return conversations, err
}

func (g *GORMCodeAssistDAO) ClaimConversation(ctx context.Context, id, userID int64,
	runToken string) error {
	now := time.Now()
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&AIConversation{}).
			Where(`id = ? AND user_id = ? AND
                (status = ? OR (status = ? AND utime < ?))`,
				id, userID, domain.AIConversationStatusActive,
				domain.AIConversationStatusRunning, now.Add(-aiConversationClaimTimeout).UnixMilli()).
			Updates(map[string]any{
				"status":    domain.AIConversationStatusRunning,
				"run_token": runToken, "utime": now.UnixMilli(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errs.ErrAIConversationBusy
		}
		return tx.Model(&AIMessage{}).
			Where("conversation_id = ? AND status = ?", id, domain.AIMessageStatusStreaming).
			Updates(map[string]any{
				"status":        domain.AIMessageStatusCancelled,
				"error_message": "AI generation lease expired", "utime": now.UnixMilli(),
			}).Error
	})
}

func (g *GORMCodeAssistDAO) ReleaseConversation(ctx context.Context, id int64,
	runToken string) error {
	return g.db.WithContext(ctx).Model(&AIConversation{}).
		Where("id = ? AND status = ? AND run_token = ?",
			id, domain.AIConversationStatusRunning, runToken).
		Updates(map[string]any{
			"status":    domain.AIConversationStatusActive,
			"run_token": "", "utime": time.Now().UnixMilli(),
		}).Error
}

func (g *GORMCodeAssistDAO) CreateMessage(ctx context.Context, message AIMessage) (AIMessage, error) {
	now := time.Now().UnixMilli()
	message.CTime, message.UTime = now, now
	err := g.db.WithContext(ctx).Create(&message).Error
	return message, err
}

func (g *GORMCodeAssistDAO) CompleteMessage(ctx context.Context, message AIMessage) error {
	result := g.db.WithContext(ctx).Model(&AIMessage{}).
		Where("id = ? AND status = ?", message.ID, domain.AIMessageStatusStreaming).
		Updates(map[string]any{
			"content": message.Content, "status": domain.AIMessageStatusCompleted,
			"provider": message.Provider, "model": message.Model,
			"input_tokens": message.InputTokens, "output_tokens": message.OutputTokens,
			"latency_millis": message.LatencyMillis, "utime": time.Now().UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("AI message state changed, ID=%d", message.ID)
	}
	return nil
}

func (g *GORMCodeAssistDAO) FailMessage(ctx context.Context, message AIMessage, status,
	errorMessage string) error {
	return g.db.WithContext(ctx).Model(&AIMessage{}).
		Where("id = ? AND status = ?", message.ID, domain.AIMessageStatusStreaming).
		Updates(map[string]any{
			"content": message.Content, "status": status, "error_message": errorMessage,
			"input_tokens": message.InputTokens, "output_tokens": message.OutputTokens,
			"latency_millis": message.LatencyMillis, "utime": time.Now().UnixMilli(),
		}).Error
}

func (g *GORMCodeAssistDAO) ListMessages(ctx context.Context, conversationID int64,
	limit int) ([]AIMessage, error) {
	var messages []AIMessage
	err := g.db.WithContext(ctx).Where("conversation_id = ?", conversationID).
		Order("id DESC").Limit(limit).Find(&messages).Error
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, err
}

func (g *GORMCodeAssistDAO) CreateSuggestion(ctx context.Context,
	suggestion AISuggestion) (AISuggestion, error) {
	now := time.Now().UnixMilli()
	suggestion.CTime, suggestion.UTime = now, now
	err := g.db.WithContext(ctx).Create(&suggestion).Error
	return suggestion, err
}

func (g *GORMCodeAssistDAO) GetSuggestionByID(ctx context.Context, id int64) (AISuggestion, error) {
	var suggestion AISuggestion
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&suggestion).Error
	return suggestion, err
}

func (g *GORMCodeAssistDAO) ListSuggestions(ctx context.Context,
	conversationID int64) ([]AISuggestion, error) {
	var suggestions []AISuggestion
	err := g.db.WithContext(ctx).Where("conversation_id = ?", conversationID).
		Order("id ASC").Find(&suggestions).Error
	return suggestions, err
}

func (g *GORMCodeAssistDAO) ClaimSuggestion(ctx context.Context, id int64) error {
	now := time.Now()
	result := g.db.WithContext(ctx).Model(&AISuggestion{}).
		Where(`id = ? AND (status IN ? OR (status = ? AND utime < ?))`, id,
			[]string{string(domain.AISuggestionStatusDraft), string(domain.AISuggestionStatusValidated)},
			domain.AISuggestionStatusApplying, now.Add(-aiSuggestionClaimTimeout).UnixMilli()).
		Updates(map[string]any{
			"status": domain.AISuggestionStatusApplying, "utime": now.UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrAISuggestionConflict
	}
	return nil
}

func (g *GORMCodeAssistDAO) ReleaseSuggestion(ctx context.Context, id int64, status string) error {
	return g.db.WithContext(ctx).Model(&AISuggestion{}).
		Where("id = ? AND status = ?", id, domain.AISuggestionStatusApplying).
		Updates(map[string]any{"status": status, "utime": time.Now().UnixMilli()}).Error
}

func (g *GORMCodeAssistDAO) MarkSuggestionApplied(ctx context.Context, id, versionID int64) error {
	result := g.db.WithContext(ctx).Model(&AISuggestion{}).
		Where("id = ? AND status = ?", id, domain.AISuggestionStatusApplying).
		Updates(map[string]any{
			"status":             domain.AISuggestionStatusApplied,
			"applied_version_id": versionID, "utime": time.Now().UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrAISuggestionConflict
	}
	return nil
}
