package executor

type ListExecutorsReq struct {
	Limit   int64  `form:"limit"`
	Cursor  string `form:"cursor"`
	Keyword string `form:"keyword"`
}

type ListExecutorsResp struct {
	Executors  []ExecutorVO `json:"executors"`
	NextCursor string       `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}

type ExecutorVO struct {
	Name     string          `json:"name"`
	Desc     string          `json:"desc"`
	Mode     string          `json:"mode"`
	Handlers []HandlerDetail `json:"handlers"`
	Nodes    []NodeDetail    `json:"nodes"`
}

type NodeDetail struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type HandlerDetail struct {
	Name     string        `json:"name"`
	Desc     string        `json:"desc"`
	Metadata []ParameterVO `json:"metadata"`
}

type ParameterVO struct {
	Key      string               `json:"key"`
	Desc     string               `json:"desc"`
	Secret   bool                 `json:"secret"`
	Required bool                 `json:"required"`
	Bindings map[string]BindingVO `json:"bindings"`
	Default  string               `json:"default"`
}

type BindingVO struct {
	Label       string            `json:"label"`
	Placeholder string            `json:"placeholder"`
	Component   string            `json:"component"`
	Config      map[string]string `json:"config"`
}
