package codeassist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Duke1616/eiam/pkg/web/capability"
	"github.com/Duke1616/etask/internal/domain"
	codeassistSvc "github.com/Duke1616/etask/internal/service/codeassist"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
)

const streamHeartbeatInterval = 15 * time.Second

var _ ginx.Handler = (*Handler)(nil)

// Handler 提供 Codebook AI 对话和候选代码接口。
type Handler struct {
	svc codeassistSvc.Service
	capability.IRegistry
}

// NewHandler 创建 CodeAssist Web 处理器。
func NewHandler(svc codeassistSvc.Service) *Handler {
	return &Handler{
		svc:       svc,
		IRegistry: capability.NewRegistry("task", "code_assist", "脚本引擎/AI 助手"),
	}
}

func (h *Handler) PublicRoutes(_ *gin.Engine)   {}
func (h *Handler) IdentifyRoutes(_ *gin.Engine) {}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/code-assist")
	g.POST("/conversation/create", h.Capability("创建 AI 会话", "add_conversation").
		Handle(ginx.B[CreateConversationReq](h.CreateConversation)))
	g.POST("/conversation/list", h.Capability("AI 会话列表", "view").
		Handle(ginx.B[ListConversationsReq](h.ListConversations)))
	g.POST("/conversation/detail", h.Capability("AI 会话详情", "get_conversation").
		NoSync().
		Needs("task:code_assist:view").
		Handle(ginx.B[ConversationDetailReq](h.ConversationDetail)))
	g.POST("/message/stream", h.Capability("发送 AI 消息", "chat").
		Handle(ginx.B[ChatReq](h.StreamChat)))
	g.POST("/suggestion/apply", h.Capability("保存 AI 候选版本", "apply_suggestion").
		Needs("task:codebook:add_version").
		Handle(ginx.B[ApplySuggestionReq](h.ApplySuggestion)))
}

func (h *Handler) CreateConversation(ctx *ginx.Context,
	req CreateConversationReq) (ginx.Result, error) {
	conversation, err := h.svc.CreateConversation(ctx, req.ProjectID, req.Title)
	if err != nil {
		return translateError(err), err
	}
	return ginx.Result{Msg: "success", Data: toConversationVO(conversation)}, nil
}

func (h *Handler) ListConversations(ctx *ginx.Context,
	req ListConversationsReq) (ginx.Result, error) {
	conversations, err := h.svc.ListConversations(ctx, req.ProjectID)
	if err != nil {
		return translateError(err), err
	}
	result := make([]ConversationVO, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, toConversationVO(conversation))
	}
	return ginx.Result{Msg: "success", Data: ConversationListResp{Conversations: result}}, nil
}

func (h *Handler) ConversationDetail(ctx *ginx.Context,
	req ConversationDetailReq) (ginx.Result, error) {
	messages, suggestions, err := h.svc.ConversationDetail(ctx, req.ConversationID)
	if err != nil {
		return translateError(err), err
	}
	messageVOs := make([]MessageVO, 0, len(messages))
	for _, message := range messages {
		messageVOs = append(messageVOs, toMessageVO(message))
	}
	suggestionVOs := make([]SuggestionVO, 0, len(suggestions))
	for _, suggestion := range suggestions {
		suggestionVOs = append(suggestionVOs, toSuggestionVO(suggestion))
	}
	return ginx.Result{Msg: "success", Data: ConversationDetailResp{
		Messages: messageVOs, Suggestions: suggestionVOs,
	}}, nil
}

// StreamChat 将模型响应以 SSE 事件实时返回给前端。
func (h *Handler) StreamChat(ctx *ginx.Context, req ChatReq) (ginx.Result, error) {
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	requestCtx := ctx.Request.Context()
	events := make(chan codeassistSvc.StreamEvent)
	done := make(chan error, 1)
	go func() {
		done <- h.runChat(requestCtx, toChatRequest(req), func(event codeassistSvc.StreamEvent) error {
			select {
			case events <- event:
				return nil
			case <-requestCtx.Done():
				return requestCtx.Err()
			}
		})
	}()

	heartbeat := time.NewTicker(streamHeartbeatInterval)
	defer heartbeat.Stop()
	failedSent := false
	for {
		select {
		case event := <-events:
			failedSent = failedSent || event.Type == codeassistSvc.StreamEventTypeFailed
			h.writeStreamEvent(ctx, event)
		case err := <-done:
			requestCancelled := requestCtx.Err() != nil && errors.Is(err, requestCtx.Err())
			if err != nil && !requestCancelled {
				_ = ctx.Error(err)
				if !failedSent {
					h.writeStreamEvent(ctx, codeassistSvc.StreamEvent{
						Type: codeassistSvc.StreamEventTypeFailed, Err: err,
					})
				}
			}
			return ginx.Result{}, ginx.ErrNoResponse
		case <-heartbeat.C:
			ctx.SSEvent("heartbeat", gin.H{})
			ctx.Writer.Flush()
		case <-requestCtx.Done():
			return ginx.Result{}, ginx.ErrNoResponse
		}
	}
}

func (h *Handler) runChat(ctx context.Context, request domain.AIChatRequest,
	emit codeassistSvc.EventEmitter) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("AI chat execution panicked: %v", recovered)
		}
	}()
	return h.svc.Chat(ctx, request, emit)
}

func (h *Handler) writeStreamEvent(ctx *ginx.Context, event codeassistSvc.StreamEvent) {
	errorMessage := ""
	if event.Err != nil {
		errorMessage = publicStreamError(event.Err)
	}
	ctx.SSEvent(string(event.Type), StreamEventVO{
		MessageID: event.MessageID, Text: event.Text,
		InputTokens: event.Usage.InputTokens, OutputTokens: event.Usage.OutputTokens,
		Error: errorMessage,
	})
	ctx.Writer.Flush()
}

func (h *Handler) ApplySuggestion(ctx *ginx.Context, req ApplySuggestionReq) (ginx.Result, error) {
	versionID, err := h.svc.ApplySuggestion(ctx, req.ID)
	if err != nil {
		return translateError(err), err
	}
	return ginx.Result{Msg: "success", Data: ApplySuggestionResp{VersionID: versionID}}, nil
}
