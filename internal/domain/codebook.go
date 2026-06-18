package domain

import (
	"fmt"

	"github.com/Duke1616/etask/internal/errs"
)

// Codebook 表示 etask 负责维护的可执行脚本模板。
type Codebook struct {
	ID         int64
	TenantID   int64
	Name       string
	Owner      string
	Code       string
	Language   string
	Secret     string
	Identifier string
	CTime      int64
	UTime      int64
}

func (c *Codebook) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: name is empty", errs.ErrInvalidParameter)
	}
	if c.Identifier == "" {
		return fmt.Errorf("%w: identifier is empty", errs.ErrInvalidParameter)
	}
	if c.Code == "" {
		return fmt.Errorf("%w: code is empty", errs.ErrInvalidParameter)
	}
	return nil
}
