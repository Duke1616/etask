package variable

type CreateReq struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type UpdateReq struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type Page struct {
	Offset int64 `json:"offset,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
}

type ListReq struct {
	Page
	Keyword string `json:"keyword"`
}

type VariableVO struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
	CTime  int64  `json:"ctime"`
	UTime  int64  `json:"utime"`
}

type ListVariablesResp struct {
	Total     int64        `json:"total"`
	Variables []VariableVO `json:"variables"`
}
