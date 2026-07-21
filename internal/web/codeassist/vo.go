package codeassist

// CreateConversationReq 创建项目级 AI 会话。
type CreateConversationReq struct {
	ProjectID int64  `json:"project_id"`
	Title     string `json:"title"`
}

// ListConversationsReq 查询项目下的 AI 会话。
type ListConversationsReq struct {
	ProjectID int64 `json:"project_id"`
}

// ConversationVO 是前端使用的 AI 会话。
type ConversationVO struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Model string `json:"model"`
	UTime int64  `json:"utime"`
}

// ConversationListResp 返回 AI 会话分页结果。
type ConversationListResp struct {
	Conversations []ConversationVO `json:"conversations"`
}

// ConversationDetailReq 查询会话详情。
type ConversationDetailReq struct {
	ConversationID int64 `json:"conversation_id"`
}

// MessageVO 是前端使用的 AI 消息。
type MessageVO struct {
	ID            int64  `json:"id"`
	Role          string `json:"role"`
	Content       string `json:"content"`
	Status        string `json:"status"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	LatencyMillis int64  `json:"latency_millis"`
	ErrorMessage  string `json:"error_message"`
	CTime         int64  `json:"ctime"`
}

// ChatContextReq 描述编辑器附带的当前代码上下文。
type ChatContextReq struct {
	NodeID        int64  `json:"node_id"`
	BaseVersionID int64  `json:"base_version_id"`
	EditorCode    string `json:"editor_code"`
}

// ChatReq 是一次流式 AI 消息请求。
type ChatReq struct {
	ConversationID int64          `json:"conversation_id"`
	RecipeID       string         `json:"recipe_id"`
	Content        string         `json:"content"`
	Context        ChatContextReq `json:"context"`
}

// StreamEventVO 是 SSE 返回的 AI 对话事件。
type StreamEventVO struct {
	MessageID    int64  `json:"message_id,omitempty"`
	Text         string `json:"text,omitempty"`
	InputTokens  int64  `json:"input_tokens,omitempty"`
	OutputTokens int64  `json:"output_tokens,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ApplySuggestionReq 应用候选代码。
type ApplySuggestionReq struct {
	ID int64 `json:"id"`
}

// DiagnosticVO 是候选代码静态诊断结果。
type DiagnosticVO struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// SuggestionVO 是前端使用的候选代码。
type SuggestionVO struct {
	ID               int64          `json:"id"`
	MessageID        int64          `json:"message_id"`
	NodeID           int64          `json:"node_id"`
	Code             string         `json:"code"`
	Summary          string         `json:"summary"`
	Diagnostics      []DiagnosticVO `json:"diagnostics"`
	Status           string         `json:"status"`
	AppliedVersionID int64          `json:"applied_version_id"`
}

// ConversationDetailResp 返回会话中的消息和候选代码。
type ConversationDetailResp struct {
	Messages    []MessageVO    `json:"messages"`
	Suggestions []SuggestionVO `json:"suggestions"`
}

// ApplySuggestionResp 返回候选代码创建的新版本。
type ApplySuggestionResp struct {
	VersionID int64 `json:"version_id"`
}
