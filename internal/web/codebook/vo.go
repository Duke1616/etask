package codebook

type CreateReq struct {
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Code       string `json:"code"`
	Language   string `json:"language"`
	Identifier string `json:"identifier"`
}

type UpdateReq struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Code     string `json:"code"`
	Language string `json:"language"`
}

type Page struct {
	Offset int64 `json:"offset,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
}

type ListReq struct {
	Page
}

type CodebookVO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Language   string `json:"language"`
	Secret     string `json:"secret"`
	CTime      int64  `json:"ctime"`
	UTime      int64  `json:"utime"`
}

type ListCodebooksResp struct {
	Total     int64        `json:"total"`
	Codebooks []CodebookVO `json:"codebooks"`
}
