package codeassist

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/ai"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository"
	codeassistRecipe "github.com/Duke1616/etask/internal/service/codeassist/recipe"
	codebookSvc "github.com/Duke1616/etask/internal/service/codebook"
	codebookmocks "github.com/Duke1616/etask/internal/service/codebook/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func hashContent(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

type codeAssistRepositoryStub struct {
	repository.CodeAssistRepository
	conversation domain.AIConversation
	messages     []domain.AIMessage
	suggestion   domain.AISuggestion
	claimed      bool
	claimToken   string
	releaseToken string
	applied      int64
	failed       domain.AIMessage
	failStatus   domain.AIMessageStatus
}

func (r *codeAssistRepositoryStub) GetConversationByID(context.Context,
	int64) (domain.AIConversation, error) {
	return r.conversation, nil
}

func (r *codeAssistRepositoryStub) ClaimConversation(_ context.Context, _ int64, _ int64,
	runToken string) error {
	r.claimed = true
	r.claimToken = runToken
	return nil
}

func (r *codeAssistRepositoryStub) ReleaseConversation(_ context.Context, _ int64,
	runToken string) error {
	r.claimed = false
	r.releaseToken = runToken
	return nil
}

func (r *codeAssistRepositoryStub) ListMessages(context.Context, int64,
	int) ([]domain.AIMessage, error) {
	return append([]domain.AIMessage(nil), r.messages...), nil
}

func (r *codeAssistRepositoryStub) CreateMessage(_ context.Context,
	message domain.AIMessage) (domain.AIMessage, error) {
	message.ID = int64(len(r.messages) + 1)
	r.messages = append(r.messages, message)
	return message, nil
}

func (r *codeAssistRepositoryStub) CompleteMessage(_ context.Context, message domain.AIMessage) error {
	message.Status = domain.AIMessageStatusCompleted
	for index := range r.messages {
		if r.messages[index].ID == message.ID {
			r.messages[index] = message
		}
	}
	return nil
}

func (r *codeAssistRepositoryStub) FailMessage(_ context.Context, message domain.AIMessage,
	status domain.AIMessageStatus, _ string) error {
	r.failed = message
	r.failStatus = status
	return nil
}

func (r *codeAssistRepositoryStub) CreateSuggestion(_ context.Context,
	suggestion domain.AISuggestion) (domain.AISuggestion, error) {
	suggestion.ID = 31
	r.suggestion = suggestion
	return suggestion, nil
}

func (r *codeAssistRepositoryStub) GetSuggestionByID(context.Context,
	int64) (domain.AISuggestion, error) {
	return r.suggestion, nil
}

func (r *codeAssistRepositoryStub) ClaimSuggestion(context.Context, int64) error { return nil }

func (r *codeAssistRepositoryStub) ReleaseSuggestion(context.Context, int64,
	domain.AISuggestionStatus) error {
	return nil
}

func (r *codeAssistRepositoryStub) MarkSuggestionApplied(_ context.Context,
	_ int64, versionID int64) error {
	r.applied = versionID
	r.suggestion.Status = domain.AISuggestionStatusApplied
	r.suggestion.AppliedVersionID = versionID
	return nil
}

type workspaceStub struct{ codebookSvc.WorkspaceService }

func (workspaceStub) Tree(context.Context, int64) ([]domain.WorkspaceNode, error) {
	return []domain.WorkspaceNode{{
		Name: "system", RuntimePath: "system", Layer: domain.WorkspaceLayerSystem,
	}}, nil
}

type providerStub struct {
	events      []ai.Event
	lastRequest *ai.Request
}

func (providerStub) Name() string  { return "fake" }
func (providerStub) Model() string { return "fake-code-model" }
func (p providerStub) Stream(_ context.Context, request ai.Request) (ai.Stream, error) {
	if p.lastRequest != nil {
		*p.lastRequest = request
	}
	return &streamStub{events: p.events}, nil
}

type streamStub struct {
	events  []ai.Event
	current int
}

func (s *streamStub) Next() bool {
	if s.current >= len(s.events) {
		return false
	}
	s.current++
	return true
}
func (s *streamStub) Current() ai.Event { return s.events[s.current-1] }
func (s *streamStub) Err() error        { return nil }
func (s *streamStub) Close() error      { return nil }

func TestServiceChatCreatesSuggestion(t *testing.T) {
	controller := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(controller)
	repo := &codeAssistRepositoryStub{conversation: domain.AIConversation{
		ID: 1, UserID: 2, ProjectID: 3, Status: domain.AIConversationStatusActive,
	}}
	baseCode := "import sys\nprint(sys.argv[1])\n"
	baseHash := hashContent(baseCode)
	codebooks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(domain.Codebook{
		ID: 10, ProjectID: 3, Name: "task.py", Kind: domain.CodebookKindFile,
		Code: baseCode, CurrentVersionID: 20,
	}, nil)
	codebooks.EXPECT().GetVersionByID(gomock.Any(), int64(20)).Return(domain.CodebookVersion{
		ID: 20, NodeID: 10, Code: baseCode, Hash: baseHash,
	}, nil)
	var modelRequest ai.Request
	provider := providerStub{lastRequest: &modelRequest, events: []ai.Event{
		{Type: ai.EventTypeTextDelta, Text: "已经生成修复建议。"},
		{Type: ai.EventTypeToolCallStarted},
		{Type: ai.EventTypeToolCall, ToolCall: &ai.ToolCall{
			Name:      proposeCodeToolName,
			Arguments: `{"summary":"升级运行协议","code":"import json\nimport os\nwith open(os.environ['ETASK_ARGS_FILE']) as f:\n    print(json.load(f))\n"}`,
		}},
		{Type: ai.EventTypeCompleted,
			Usage: ai.Usage{InputTokens: 10, OutputTokens: 20}},
	}}
	service := NewService(repo, codebooks, workspaceStub{}, provider, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithTenantID(t.Context(), 1)
	ctx = ctxutil.WithUserID(ctx, 2)
	events := make([]StreamEvent, 0)

	err := service.Chat(ctx, domain.AIChatRequest{
		ConversationID: 1, Content: "升级当前脚本",
		Context: domain.AIChatContext{
			NodeID: 10, BaseVersionID: 20, EditorCode: baseCode,
		},
	}, func(event StreamEvent) error {
		events = append(events, event)
		return nil
	})

	require.NoError(t, err)
	require.False(t, repo.claimed)
	require.Equal(t, int64(31), repo.suggestion.ID)
	require.Equal(t, domain.AISuggestionStatusValidated, repo.suggestion.Status)
	require.Equal(t, codeassistRecipe.GeneralID, repo.suggestion.RecipeID)
	require.Equal(t, "1", repo.suggestion.RecipeVersion)
	require.Len(t, modelRequest.Tools, 1)
	require.Equal(t, "fake", repo.messages[1].Provider)
	require.Equal(t, repo.claimToken, repo.releaseToken)
	require.Equal(t, []StreamEventType{
		StreamEventTypeStarted,
		StreamEventTypeDelta,
		StreamEventTypeProgress,
		StreamEventTypeProgress,
		StreamEventTypeCompleted,
	}, streamEventTypes(events))
}

func TestServiceChatRejectsMultipleSuggestionsBeforePersisting(t *testing.T) {
	controller := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(controller)
	repo := &codeAssistRepositoryStub{conversation: domain.AIConversation{
		ID: 1, UserID: 2, ProjectID: 3, Status: domain.AIConversationStatusActive,
	}}
	baseCode := "print('old')\n"
	codebooks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(domain.Codebook{
		ID: 10, ProjectID: 3, Name: "task.py", Kind: domain.CodebookKindFile,
		Code: baseCode, CurrentVersionID: 20,
	}, nil)
	codebooks.EXPECT().GetVersionByID(gomock.Any(), int64(20)).Return(domain.CodebookVersion{
		ID: 20, NodeID: 10, Code: baseCode, Hash: hashContent(baseCode),
	}, nil)
	provider := providerStub{events: []ai.Event{
		{Type: ai.EventTypeToolCall, ToolCall: &ai.ToolCall{
			Name: proposeCodeToolName, Arguments: `{"summary":"first","code":"print(1)\n"}`,
		}},
		{Type: ai.EventTypeToolCall, ToolCall: &ai.ToolCall{
			Name: proposeCodeToolName, Arguments: `{"summary":"second","code":"print(2)\n"}`,
		}},
		{Type: ai.EventTypeCompleted},
	}}
	service := NewService(repo, codebooks, workspaceStub{}, provider, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithUserID(ctxutil.WithTenantID(t.Context(), 1), 2)

	err := service.Chat(ctx, domain.AIChatRequest{
		ConversationID: 1, RecipeID: codeassistRecipe.GeneralID, Content: "生成两个方案",
		Context: domain.AIChatContext{
			NodeID: 10, BaseVersionID: 20, EditorCode: baseCode,
		},
	}, func(StreamEvent) error { return nil })

	require.EqualError(t, err, "AI response contains multiple code suggestions")
	require.Zero(t, repo.suggestion.ID)
	require.Equal(t, domain.AIMessageStatusFailed, repo.failStatus)
}

func TestServiceChatRejectsEmptyCompletedResponse(t *testing.T) {
	repo := &codeAssistRepositoryStub{conversation: domain.AIConversation{
		ID: 1, UserID: 2, ProjectID: 3, Status: domain.AIConversationStatusActive,
	}}
	provider := providerStub{events: []ai.Event{{
		Type:  ai.EventTypeCompleted,
		Usage: ai.Usage{InputTokens: 943, OutputTokens: 8192},
	}}}
	service := NewService(repo, nil, workspaceStub{}, provider, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithUserID(ctxutil.WithTenantID(t.Context(), 1), 2)

	err := service.Chat(ctx, domain.AIChatRequest{
		ConversationID: 1, RecipeID: codeassistRecipe.GeneralID, Content: "修改代码",
	}, func(StreamEvent) error { return nil })

	require.EqualError(t, err, "模型未返回可展示的文本或代码候选")
	require.Equal(t, domain.AIMessageStatusFailed, repo.failStatus)
	require.Equal(t, int64(943), repo.failed.InputTokens)
	require.Equal(t, int64(8192), repo.failed.OutputTokens)
}

func TestServiceChatRequiresRecipeFileContext(t *testing.T) {
	repo := &codeAssistRepositoryStub{conversation: domain.AIConversation{
		ID: 1, UserID: 2, ProjectID: 3, Status: domain.AIConversationStatusActive,
	}}
	service := NewService(repo, nil, workspaceStub{}, providerStub{}, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithUserID(ctxutil.WithTenantID(t.Context(), 1), 2)

	err := service.Chat(ctx, domain.AIChatRequest{
		ConversationID: 1, RecipeID: "codebook.edit", Content: "修改代码",
	}, func(StreamEvent) error { return nil })

	require.ErrorContains(t, err, "AI recipe requires a Codebook file context")
	require.False(t, repo.claimed)
	require.Empty(t, repo.messages)
}

func TestServiceChatSettlesInterruptedMessage(t *testing.T) {
	testCases := []struct {
		name   string
		before func(StreamEvent, context.CancelFunc) error
		after  func(*testing.T, *codeAssistRepositoryStub, error)
	}{
		{
			name: "连接在部分输出后断开",
			before: func(event StreamEvent, cancel context.CancelFunc) error {
				if event.Type != StreamEventTypeDelta {
					return nil
				}
				cancel()
				return context.Canceled
			},
			after: func(t *testing.T, repo *codeAssistRepositoryStub, err error) {
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, domain.AIMessageStatusCancelled, repo.failStatus)
				require.Equal(t, "部分回复", repo.failed.Content)
			},
		},
		{
			name: "开始事件写入失败",
			before: func(event StreamEvent, _ context.CancelFunc) error {
				if event.Type == StreamEventTypeStarted {
					return errors.New("SSE writer is unavailable")
				}
				return nil
			},
			after: func(t *testing.T, repo *codeAssistRepositoryStub, err error) {
				require.EqualError(t, err, "SSE writer is unavailable")
				require.Equal(t, domain.AIMessageStatusFailed, repo.failStatus)
				require.Empty(t, repo.failed.Content)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &codeAssistRepositoryStub{conversation: domain.AIConversation{
				ID: 1, UserID: 2, ProjectID: 3, Status: domain.AIConversationStatusActive,
			}}
			provider := providerStub{events: []ai.Event{
				{Type: ai.EventTypeTextDelta, Text: "部分回复"},
				{Type: ai.EventTypeCompleted},
			}}
			service := NewService(repo, nil, workspaceStub{}, provider, codeassistRecipe.NewCatalog())
			baseCtx, cancel := context.WithCancel(t.Context())
			ctx := ctxutil.WithUserID(ctxutil.WithTenantID(baseCtx, 1), 2)
			t.Cleanup(cancel)

			err := service.Chat(ctx, domain.AIChatRequest{
				ConversationID: 1, RecipeID: codeassistRecipe.GeneralID, Content: "分析脚本",
			}, func(event StreamEvent) error {
				return testCase.before(event, cancel)
			})

			testCase.after(t, repo, err)
			require.False(t, repo.claimed)
		})
	}
}

func TestServiceApplySuggestionDoesNotDependOnHistoricalRecipe(t *testing.T) {
	controller := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(controller)
	baseCode := "print('old')\n"
	newCode := "print('new')\n"
	repo := &codeAssistRepositoryStub{
		conversation: domain.AIConversation{ID: 1, UserID: 2, ProjectID: 3},
		suggestion: domain.AISuggestion{
			ID: 31, ConversationID: 1, ProjectID: 3, NodeID: 10,
			BaseVersionID: 20, BaseHash: hashContent(baseCode),
			RecipeID: "codebook.removed-recipe", RecipeVersion: "7",
			Language: "python", Code: newCode, Status: domain.AISuggestionStatusValidated,
		},
	}
	codebooks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(domain.Codebook{
		ID: 10, ProjectID: 3, Name: "task.py", Kind: domain.CodebookKindFile,
		CurrentVersionID: 20,
	}, nil)
	codebooks.EXPECT().GetVersionByID(gomock.Any(), int64(20)).Return(domain.CodebookVersion{
		ID: 20, NodeID: 10, Code: baseCode, Hash: hashContent(baseCode),
	}, nil)
	codebooks.EXPECT().CreateVersion(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, version domain.CodebookVersionCreate) (int64, error) {
			require.Equal(t, int64(10), version.NodeID)
			require.Equal(t, int64(20), version.ExpectedCurrentVersionID)
			require.Equal(t, "ai-suggestion:31", version.SourceKey)
			require.Equal(t, newCode, version.Code)
			return 40, nil
		})
	service := NewService(repo, codebooks, workspaceStub{}, providerStub{}, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithUserID(ctxutil.WithTenantID(t.Context(), 1), 2)

	versionID, err := service.ApplySuggestion(ctx, 31)

	require.NoError(t, err)
	require.Equal(t, int64(40), versionID)
	require.Equal(t, int64(40), repo.applied)
}

func TestServiceApplySuggestionRetriesApplyingSuggestion(t *testing.T) {
	controller := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(controller)
	code := "print('new')\n"
	repo := &codeAssistRepositoryStub{
		conversation: domain.AIConversation{ID: 1, UserID: 2, ProjectID: 3},
		suggestion: domain.AISuggestion{
			ID: 31, ConversationID: 1, ProjectID: 3, NodeID: 10,
			BaseVersionID: 20, BaseHash: hashContent("print('old')\n"),
			RecipeID: codeassistRecipe.GeneralID, RecipeVersion: "1",
			Language: "python", Code: code, Status: domain.AISuggestionStatusApplying,
		},
	}
	codebooks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(domain.Codebook{
		ID: 10, ProjectID: 3, Name: "task.py", Kind: domain.CodebookKindFile,
		CurrentVersionID: 20,
	}, nil)
	codebooks.EXPECT().GetVersionByID(gomock.Any(), int64(20)).Return(domain.CodebookVersion{
		ID: 20, NodeID: 10, Hash: hashContent("print('old')\n"),
	}, nil)
	codebooks.EXPECT().CreateVersion(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, version domain.CodebookVersionCreate) (int64, error) {
			require.Equal(t, "ai-suggestion:31", version.SourceKey)
			return 40, nil
		})
	service := NewService(repo, codebooks, workspaceStub{}, providerStub{}, codeassistRecipe.NewCatalog())
	ctx := ctxutil.WithUserID(ctxutil.WithTenantID(t.Context(), 1), 2)

	versionID, err := service.ApplySuggestion(ctx, 31)

	require.NoError(t, err)
	require.Equal(t, int64(40), versionID)
	require.Equal(t, int64(40), repo.applied)
}

func streamEventTypes(events []StreamEvent) []StreamEventType {
	result := make([]StreamEventType, 0, len(events))
	for _, event := range events {
		result = append(result, event.Type)
	}
	return result
}
