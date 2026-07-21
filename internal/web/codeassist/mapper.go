package codeassist

import "github.com/Duke1616/etask/internal/domain"

func toChatRequest(req ChatReq) domain.AIChatRequest {
	return domain.AIChatRequest{
		ConversationID: req.ConversationID, RecipeID: req.RecipeID, Content: req.Content,
		Context: domain.AIChatContext{
			NodeID: req.Context.NodeID, BaseVersionID: req.Context.BaseVersionID,
			EditorCode: req.Context.EditorCode,
		},
	}
}

func toConversationVO(source domain.AIConversation) ConversationVO {
	return ConversationVO{
		ID: source.ID, Title: source.Title, Model: source.Model, UTime: source.UTime,
	}
}

func toMessageVO(source domain.AIMessage) MessageVO {
	return MessageVO{
		ID:   source.ID,
		Role: string(source.Role), Content: source.Content, Status: string(source.Status),
		InputTokens: source.InputTokens, OutputTokens: source.OutputTokens,
		LatencyMillis: source.LatencyMillis, ErrorMessage: source.ErrorMessage,
		CTime: source.CTime,
	}
}

func toSuggestionVO(source domain.AISuggestion) SuggestionVO {
	diagnostics := make([]DiagnosticVO, 0, len(source.Diagnostics))
	for _, diagnostic := range source.Diagnostics {
		diagnostics = append(diagnostics, DiagnosticVO{
			Severity: string(diagnostic.Severity), Code: diagnostic.Code, Message: diagnostic.Message,
		})
	}
	return SuggestionVO{
		ID: source.ID, MessageID: source.MessageID, NodeID: source.NodeID,
		Code: source.Code, Summary: source.Summary,
		Diagnostics: diagnostics, Status: string(source.Status),
		AppliedVersionID: source.AppliedVersionID,
	}
}
