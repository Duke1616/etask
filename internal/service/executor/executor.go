package executor

import (
	"context"
	"strings"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const defaultLimit int64 = 20

type Service interface {
	List(ctx context.Context, limit int64, cursor string, keyword string) (domain.ExecutorList, error)
}

type service struct {
	store executorStore
}

func NewService(client *clientv3.Client) Service {
	return &service{store: newEtcdExecutorStore(client)}
}

func (s *service) List(ctx context.Context, limit int64, cursor string, keyword string) (domain.ExecutorList, error) {
	keyword = strings.TrimSpace(keyword)
	page, err := s.store.ListGroups(ctx, normalizeLimit(limit), cursor, func(group executorGroup) bool {
		return matchKeyword(toExecutor(group), keyword)
	})
	if err != nil {
		return domain.ExecutorList{}, err
	}

	return domain.ExecutorList{
		Executors:  lo.Map(page.Groups, func(group executorGroup, _ int) domain.Executor { return toExecutor(group) }),
		NextCursor: page.NextCursor,
	}, nil
}

func normalizeLimit(limit int64) int64 {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}

func matchKeyword(exec domain.Executor, keyword string) bool {
	if keyword == "" {
		return true
	}
	keyword = strings.ToLower(keyword)
	if strings.Contains(strings.ToLower(exec.Name), keyword) ||
		strings.Contains(strings.ToLower(exec.Desc), keyword) ||
		strings.Contains(strings.ToLower(exec.Mode), keyword) {
		return true
	}
	return lo.SomeBy(exec.Handlers, func(handler domain.ExecutorHandler) bool {
		return strings.Contains(strings.ToLower(handler.Name), keyword) ||
			strings.Contains(strings.ToLower(handler.Desc), keyword)
	})
}
