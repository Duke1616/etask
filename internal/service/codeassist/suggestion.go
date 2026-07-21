package codeassist

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/Duke1616/etask/internal/service/codeassist/recipe"
)

func (s *service) getSuggestion(ctx context.Context, id int64) (domain.AISuggestion, error) {
	if id <= 0 {
		return domain.AISuggestion{}, fmt.Errorf("%w: invalid AI suggestion ID", errs.ErrInvalidParameter)
	}
	suggestion, err := s.repo.GetSuggestionByID(ctx, id)
	if err != nil {
		return domain.AISuggestion{}, err
	}
	conversation, err := s.userConversation(ctx, suggestion.ConversationID)
	if err != nil || conversation.ProjectID != suggestion.ProjectID {
		return domain.AISuggestion{}, fmt.Errorf("AI suggestion is not accessible")
	}
	return suggestion, nil
}

func (s *service) ApplySuggestion(ctx context.Context, id int64) (int64, error) {
	suggestion, err := s.getSuggestion(ctx, id)
	if err != nil {
		return 0, err
	}
	if suggestion.Status == domain.AISuggestionStatusApplied {
		return suggestion.AppliedVersionID, nil
	}
	if suggestion.Status != domain.AISuggestionStatusDraft &&
		suggestion.Status != domain.AISuggestionStatusValidated &&
		suggestion.Status != domain.AISuggestionStatusApplying {
		return 0, fmt.Errorf("%w: AI suggestion cannot be applied in status %s",
			errs.ErrAISuggestionConflict, suggestion.Status)
	}
	node, err := s.codebooks.GetByID(ctx, suggestion.NodeID)
	if err != nil {
		return 0, err
	}
	if node.ProjectID != suggestion.ProjectID {
		return 0, fmt.Errorf("AI suggestion project changed")
	}
	base, err := s.codebooks.GetVersionByID(ctx, suggestion.BaseVersionID)
	if err != nil {
		return 0, err
	}
	if base.NodeID != node.ID || base.Hash != suggestion.BaseHash {
		return 0, errs.ErrCodebookVersionConflict
	}
	diagnostics := validateCandidate(ctx, suggestion.Language, suggestion.Code)
	if hasDiagnosticErrors(diagnostics) {
		return 0, fmt.Errorf("AI suggestion validation failed")
	}
	applyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	if err = s.repo.ClaimSuggestion(applyCtx, suggestion.ID); err != nil {
		return 0, err
	}
	releaseStatus := suggestion.Status
	if releaseStatus == domain.AISuggestionStatusApplying {
		releaseStatus = domain.AISuggestionStatusValidated
	}
	releaseClaim := true
	defer func() {
		if releaseClaim {
			_ = s.repo.ReleaseSuggestion(applyCtx, suggestion.ID, releaseStatus)
		}
	}()
	message := fmt.Sprintf("AI suggestion #%d", suggestion.ID)
	if suggestion.Summary != "" {
		message += ": " + suggestion.Summary
	}
	versionID, err := s.codebooks.CreateVersion(applyCtx, domain.CodebookVersionCreate{
		NodeID: node.ID, ExpectedCurrentVersionID: suggestion.BaseVersionID,
		Code: suggestion.Code, Message: message,
		SourceKey: fmt.Sprintf("ai-suggestion:%d", suggestion.ID),
	})
	if err != nil {
		return 0, err
	}
	if err = s.repo.MarkSuggestionApplied(applyCtx, suggestion.ID, versionID); err != nil {
		latest, latestErr := s.getSuggestion(applyCtx, suggestion.ID)
		if latestErr == nil && latest.Status == domain.AISuggestionStatusApplied &&
			latest.AppliedVersionID == versionID {
			releaseClaim = false
			return versionID, nil
		}
		return 0, err
	}
	releaseClaim = false
	return versionID, nil
}

type proposalArguments struct {
	Summary string `json:"summary"`
	Code    string `json:"code"`
}

func (s *service) createSuggestion(ctx context.Context, conversation domain.AIConversation,
	messageID int64, prepared preparedContext, selectedRecipe recipe.Definition,
	arguments string) (domain.AISuggestion, error) {
	if prepared.node.ID == 0 {
		return domain.AISuggestion{}, fmt.Errorf("AI proposed code without a file context")
	}
	var proposal proposalArguments
	if err := json.Unmarshal([]byte(arguments), &proposal); err != nil {
		return domain.AISuggestion{}, fmt.Errorf("invalid AI code proposal: %w", err)
	}
	if len(proposal.Code) > maxEditorCodeLength {
		return domain.AISuggestion{}, fmt.Errorf("AI suggestion code is too large")
	}
	language, err := scriptLanguage(prepared.node.Name)
	if err != nil {
		return domain.AISuggestion{}, err
	}
	if !selectedRecipe.AllowsCodeSuggestion {
		return domain.AISuggestion{}, fmt.Errorf("AI recipe does not allow code suggestions")
	}
	diagnostics := validateCandidate(ctx, language, proposal.Code)
	status := domain.AISuggestionStatusValidated
	if hasDiagnosticErrors(diagnostics) {
		status = domain.AISuggestionStatusDraft
	}
	suggestion := domain.AISuggestion{
		ConversationID: conversation.ID, MessageID: messageID,
		ProjectID: conversation.ProjectID, NodeID: prepared.node.ID,
		BaseVersionID: prepared.base.ID, BaseHash: prepared.base.Hash,
		RecipeID: selectedRecipe.ID, RecipeVersion: selectedRecipe.Version,
		Language: language, Code: proposal.Code,
		Summary: proposal.Summary, Diagnostics: diagnostics, Status: status,
	}
	if err = suggestion.Prepare(); err != nil {
		return domain.AISuggestion{}, err
	}
	return s.repo.CreateSuggestion(ctx, suggestion)
}

func scriptLanguage(name string) (string, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".py":
		return "python", nil
	case ".sh", ".bash":
		return "shell", nil
	default:
		return "", fmt.Errorf("unsupported AI script type: %s", name)
	}
}
