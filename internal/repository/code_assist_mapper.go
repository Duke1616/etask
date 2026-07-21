package repository

import (
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/Duke1616/etask/pkg/sqlx"
)

func toAIConversationEntity(source domain.AIConversation) dao.AIConversation {
	return dao.AIConversation{
		ID: source.ID, TenantID: source.TenantID, UserID: source.UserID,
		ProjectID: source.ProjectID, Title: source.Title, Provider: source.Provider,
		Model: source.Model, Status: string(source.Status), CTime: source.CTime, UTime: source.UTime,
	}
}

func toAIConversationDomain(source dao.AIConversation) domain.AIConversation {
	return domain.AIConversation{
		ID: source.ID, TenantID: source.TenantID, UserID: source.UserID,
		ProjectID: source.ProjectID, Title: source.Title, Provider: source.Provider,
		Model: source.Model, Status: domain.AIConversationStatus(source.Status),
		CTime: source.CTime, UTime: source.UTime,
	}
}

func toAIMessageEntity(source domain.AIMessage) dao.AIMessage {
	return dao.AIMessage{
		ID: source.ID, TenantID: source.TenantID, ConversationID: source.ConversationID,
		Role: string(source.Role), Content: source.Content, Status: string(source.Status),
		Provider: source.Provider, Model: source.Model,
		RecipeID: source.RecipeID, RecipeVersion: source.RecipeVersion,
		InputTokens: source.InputTokens, OutputTokens: source.OutputTokens,
		LatencyMillis: source.LatencyMillis, ErrorMessage: source.ErrorMessage,
		CTime: source.CTime, UTime: source.UTime,
	}
}

func toAIMessageDomain(source dao.AIMessage) domain.AIMessage {
	return domain.AIMessage{
		ID: source.ID, TenantID: source.TenantID, ConversationID: source.ConversationID,
		Role: domain.AIMessageRole(source.Role), Content: source.Content,
		Status: domain.AIMessageStatus(source.Status), Provider: source.Provider, Model: source.Model,
		RecipeID: source.RecipeID, RecipeVersion: source.RecipeVersion,
		InputTokens: source.InputTokens, OutputTokens: source.OutputTokens,
		LatencyMillis: source.LatencyMillis, ErrorMessage: source.ErrorMessage,
		CTime: source.CTime, UTime: source.UTime,
	}
}

func toAISuggestionEntity(source domain.AISuggestion) dao.AISuggestion {
	return dao.AISuggestion{
		ID: source.ID, TenantID: source.TenantID, ConversationID: source.ConversationID,
		MessageID: source.MessageID, ProjectID: source.ProjectID, NodeID: source.NodeID,
		BaseVersionID: source.BaseVersionID, BaseHash: source.BaseHash,
		RecipeID: source.RecipeID, RecipeVersion: source.RecipeVersion,
		Language: source.Language, Code: source.Code, Summary: source.Summary,
		Diagnostics: sqlx.JSONColumn[[]domain.AIDiagnostic]{
			Val: source.Diagnostics, Valid: source.Diagnostics != nil,
		},
		Status: string(source.Status), AppliedVersionID: source.AppliedVersionID,
		CTime: source.CTime, UTime: source.UTime,
	}
}

func toAISuggestionDomain(source dao.AISuggestion) domain.AISuggestion {
	return domain.AISuggestion{
		ID: source.ID, TenantID: source.TenantID, ConversationID: source.ConversationID,
		MessageID: source.MessageID, ProjectID: source.ProjectID, NodeID: source.NodeID,
		BaseVersionID: source.BaseVersionID, BaseHash: source.BaseHash,
		RecipeID: source.RecipeID, RecipeVersion: source.RecipeVersion,
		Language: source.Language, Code: source.Code, Summary: source.Summary,
		Diagnostics: source.Diagnostics.Val, Status: domain.AISuggestionStatus(source.Status),
		AppliedVersionID: source.AppliedVersionID, CTime: source.CTime, UTime: source.UTime,
	}
}
