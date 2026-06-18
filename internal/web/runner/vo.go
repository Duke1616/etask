package runner

type RegisterRunnerReq struct {
	Name           string     `json:"name"`
	CodebookUID    string     `json:"codebook_uid"`
	CodebookSecret string     `json:"codebook_secret"`
	Kind           string     `json:"kind"`
	Target         string     `json:"target"`
	Handler        string     `json:"handler"`
	Tags           []string   `json:"tags"`
	Desc           string     `json:"desc"`
	Variables      []Variable `json:"variables"`
}

type UpdateRunnerReq struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	CodebookUID    string     `json:"codebook_uid"`
	CodebookSecret string     `json:"codebook_secret"`
	Kind           string     `json:"kind"`
	Target         string     `json:"target"`
	Handler        string     `json:"handler"`
	Tags           []string   `json:"tags"`
	Desc           string     `json:"desc"`
	Variables      []Variable `json:"variables"`
}

type ListByCodebookUIDReq struct {
	Page
	CodebookUID string `json:"codebook_uid"`
	Keyword     string `json:"keyword"`
	Kind        string `json:"kind"`
}

type ListRunnerByIDsReq struct {
	IDs []int64 `json:"ids"`
}

type Variable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type Page struct {
	Offset int64 `json:"offset,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
}

type ListRunnerReq struct {
	Page
	Keyword string `json:"keyword"`
	Kind    string `json:"kind"`
}

type RunnerVO struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	CodebookUID string     `json:"codebook_uid"`
	Target      string     `json:"target"`
	Handler     string     `json:"handler"`
	Tags        []string   `json:"tags"`
	Variables   []Variable `json:"variables"`
	Desc        string     `json:"desc"`
	CTime       int64      `json:"ctime"`
	UTime       int64      `json:"utime"`
}

type ListRunnersResp struct {
	Total   int64      `json:"total"`
	Runners []RunnerVO `json:"runners"`
}

type TagDetail struct {
	Tag     string `json:"tag"`
	Kind    string `json:"kind"`
	Target  string `json:"target"`
	Handler string `json:"handler"`
}

type RunnerTags struct {
	CodebookUID string      `json:"codebook_uid"`
	Tags        []TagDetail `json:"tags"`
}

type ListRunnerTagsResp struct {
	RunnerTags []RunnerTags `json:"runner_tags"`
}
