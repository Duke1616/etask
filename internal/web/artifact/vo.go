package artifact

type PublishReq struct {
	Scope     string `json:"scope"`
	ProjectID int64  `json:"project_id"`
	Message   string `json:"message"`
}

type ActivateReq struct {
	Scope     string `json:"scope"`
	ProjectID int64  `json:"project_id"`
	ID        int64  `json:"id"`
}

type ListReq struct {
	Offset    int64  `json:"offset,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
	Scope     string `json:"scope"`
	ProjectID int64  `json:"project_id"`
}

type StatusReq struct {
	Scope     string `json:"scope"`
	ProjectID int64  `json:"project_id"`
}

type Release struct {
	ID             int64  `json:"id"`
	TenantID       int64  `json:"tenant_id"`
	Scope          string `json:"scope"`
	ProjectID      int64  `json:"project_id"`
	SourceRevision int64  `json:"source_revision"`
	Digest         string `json:"digest"`
	BlobChecksum   string `json:"blob_checksum"`
	Size           int64  `json:"size"`
	Format         string `json:"format"`
	FormatVersion  int32  `json:"format_version"`
	Message        string `json:"message"`
	AuthorUserID   int64  `json:"author_user_id"`
	Active         bool   `json:"active"`
	CTime          int64  `json:"ctime"`
}

type Status struct {
	Scope          string   `json:"scope"`
	ProjectID      int64    `json:"project_id"`
	SourceRevision int64    `json:"source_revision"`
	PendingChanges bool     `json:"pending_changes"`
	Active         *Release `json:"active"`
}

type ListResp struct {
	Total    int64     `json:"total"`
	Releases []Release `json:"releases"`
}
