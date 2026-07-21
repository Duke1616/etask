package codeassist

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/ai"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	"github.com/google/uuid"
)

func (s *service) Chat(ctx context.Context, request domain.AIChatRequest, emit EventEmitter) error {
	if emit == nil {
		return fmt.Errorf("%w: AI event emitter is required", errs.ErrInvalidParameter)
	}
	request.Content = strings.TrimSpace(request.Content)
	if request.ConversationID <= 0 || request.Content == "" || len(request.Content) > maxUserMessageLength {
		return fmt.Errorf("%w: invalid AI chat request", errs.ErrInvalidParameter)
	}
	selectedRecipe, err := s.recipes.Get(request.RecipeID)
	if err != nil {
		return err
	}
	conversation, err := s.userConversation(ctx, request.ConversationID)
	if err != nil {
		return err
	}
	userID := ctxutil.GetUserID(ctx).Int64()
	runToken := uuid.NewString()
	if err = s.repo.ClaimConversation(ctx, conversation.ID, userID, runToken); err != nil {
		return err
	}
	defer func() {
		settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), messageSettleTimeout)
		defer cancel()
		_ = s.repo.ReleaseConversation(settleCtx, conversation.ID, runToken)
	}()

	chatContext, err := s.prepareContext(ctx, conversation, request.Context, selectedRecipe)
	if err != nil {
		return err
	}
	history, err := s.repo.ListMessages(ctx, conversation.ID, messageHistoryLimit)
	if err != nil {
		return err
	}
	if _, err = s.repo.CreateMessage(ctx, domain.AIMessage{
		ConversationID: conversation.ID, Role: domain.AIMessageRoleUser,
		Content: request.Content, Status: domain.AIMessageStatusCompleted,
		RecipeID: selectedRecipe.ID, RecipeVersion: selectedRecipe.Version,
	}); err != nil {
		return err
	}
	assistantMessage, err := s.repo.CreateMessage(ctx, domain.AIMessage{
		ConversationID: conversation.ID, Role: domain.AIMessageRoleAssistant,
		Status:   domain.AIMessageStatusStreaming,
		Provider: s.provider.Name(), Model: s.provider.Model(),
		RecipeID: selectedRecipe.ID, RecipeVersion: selectedRecipe.Version,
	})
	if err != nil {
		return err
	}
	startedAt := time.Now()
	var content strings.Builder
	// 请求断开后仍用独立上下文收敛消息和会话状态，避免遗留 STREAMING 记录。
	fail := func(cause error) error {
		settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), messageSettleTimeout)
		defer cancel()
		status := domain.AIMessageStatusFailed
		if ctx.Err() != nil {
			status = domain.AIMessageStatusCancelled
		}
		assistantMessage.Content = content.String()
		assistantMessage.LatencyMillis = time.Since(startedAt).Milliseconds()
		_ = s.repo.FailMessage(settleCtx, assistantMessage, status, cause.Error())
		_ = emit(StreamEvent{
			Type: StreamEventTypeFailed, MessageID: assistantMessage.ID, Err: cause,
		})
		return cause
	}
	if err = emit(StreamEvent{Type: StreamEventTypeStarted, MessageID: assistantMessage.ID}); err != nil {
		return fail(err)
	}

	modelRequest := ai.Request{
		Instructions: buildInstructions(selectedRecipe.Instructions),
		Input:        buildPrompt(history, request.Content, chatContext),
		UserKey:      fmt.Sprintf("%d:%d", ctxutil.GetTenantID(ctx).Int64(), userID),
	}
	if chatContext.node.ID > 0 && selectedRecipe.AllowsCodeSuggestion {
		modelRequest.Tools = []ai.Tool{proposalTool()}
	}
	stream, err := s.provider.Stream(ctx, modelRequest)
	if err != nil {
		return fail(err)
	}
	defer stream.Close()

	var finalEvent ai.Event
	var proposal string
	completed, proposalReceived := false, false
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case ai.EventTypeTextDelta:
			content.WriteString(event.Text)
			if err = emit(StreamEvent{Type: StreamEventTypeDelta,
				MessageID: assistantMessage.ID, Text: event.Text}); err != nil {
				return fail(err)
			}
		case ai.EventTypeToolCallStarted:
			if err = emit(StreamEvent{Type: StreamEventTypeProgress,
				MessageID: assistantMessage.ID, Text: "正在生成候选代码"}); err != nil {
				return fail(err)
			}
		case ai.EventTypeToolCall:
			if event.ToolCall == nil || event.ToolCall.Name != proposeCodeToolName {
				continue
			}
			if proposalReceived {
				return fail(fmt.Errorf("AI response contains multiple code suggestions"))
			}
			proposal, proposalReceived = event.ToolCall.Arguments, true
			if err = emit(StreamEvent{Type: StreamEventTypeProgress,
				MessageID: assistantMessage.ID, Text: "正在校验候选代码"}); err != nil {
				return fail(err)
			}
		case ai.EventTypeCompleted:
			completed, finalEvent = true, event
		case ai.EventTypeFailed:
			if event.Err == nil {
				event.Err = fmt.Errorf("AI response failed")
			}
			return fail(event.Err)
		}
	}
	if err = stream.Err(); err != nil {
		return fail(err)
	}
	if !completed {
		return fail(fmt.Errorf("AI response ended without completion"))
	}
	assistantMessage.InputTokens = finalEvent.Usage.InputTokens
	assistantMessage.OutputTokens = finalEvent.Usage.OutputTokens
	if content.Len() == 0 && !proposalReceived {
		return fail(fmt.Errorf("模型未返回可展示的文本或代码候选"))
	}
	if proposalReceived {
		_, suggestionErr := s.createSuggestion(ctx, conversation,
			assistantMessage.ID, chatContext, selectedRecipe, proposal)
		if suggestionErr != nil {
			return fail(suggestionErr)
		}
	}

	assistantMessage.Content = content.String()
	if assistantMessage.Content == "" && proposalReceived {
		assistantMessage.Content = "已生成代码建议。"
	}
	assistantMessage.LatencyMillis = time.Since(startedAt).Milliseconds()
	if err = s.repo.CompleteMessage(ctx, assistantMessage); err != nil {
		return fail(err)
	}
	return emit(StreamEvent{
		Type: StreamEventTypeCompleted, MessageID: assistantMessage.ID, Usage: finalEvent.Usage,
	})
}
