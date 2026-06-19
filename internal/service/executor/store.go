package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/Duke1616/etask/pkg/grpc/registry/indexer"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type executorStore interface {
	ListGroups(ctx context.Context, limit int64, cursor string, filter executorGroupFilter) (executorGroupPage, error)
}

type executorGroupFilter func(group executorGroup) bool

type etcdExecutorStore struct {
	client *clientv3.Client
}

type executorGroupPage struct {
	Groups     []executorGroup
	NextCursor string
}

type executorGroup struct {
	Name      string
	Instances []serviceInstance
}

type serviceInstance = registry.ServiceInstance

func newEtcdExecutorStore(client *clientv3.Client) executorStore {
	return &etcdExecutorStore{client: client}
}

func (s *etcdExecutorStore) ListGroups(ctx context.Context, limit int64, cursor string, filter executorGroupFilter) (executorGroupPage, error) {
	prefix := executorIndexPrefix("")
	startKey := executorStartKey(cursor)

	groups := make([]executorGroup, 0, limit+1)
	for int64(len(groups)) < limit+1 {
		group, nextStartKey, ok, err := s.nextMatchedGroup(ctx, prefix, startKey, filter)
		if err != nil {
			return executorGroupPage{}, err
		}
		if !ok {
			break
		}
		startKey = nextStartKey
		groups = append(groups, group)
	}

	nextCursor := ""
	if int64(len(groups)) > limit {
		nextCursor = groups[limit-1].Name
		groups = groups[:limit]
	}

	return executorGroupPage{
		Groups:     groups,
		NextCursor: nextCursor,
	}, nil
}

func (s *etcdExecutorStore) nextMatchedGroup(ctx context.Context, prefix string, startKey string, filter executorGroupFilter) (executorGroup, string, bool, error) {
	for {
		name, ok, err := s.nextGroupName(ctx, prefix, startKey)
		if err != nil {
			return executorGroup{}, "", false, err
		}
		if !ok {
			return executorGroup{}, "", false, nil
		}

		nextStartKey := clientv3.GetPrefixRangeEnd(executorGroupPrefix(name))
		instances, err := s.listGroupInstances(ctx, name)
		if err != nil {
			return executorGroup{}, "", false, err
		}
		group := executorGroup{Name: name, Instances: instances}
		if filter == nil || filter(group) {
			return group, nextStartKey, true, nil
		}
		startKey = nextStartKey
	}
}

func executorStartKey(cursor string) string {
	if cursor == "" {
		return executorIndexPrefix("")
	}
	return clientv3.GetPrefixRangeEnd(executorGroupPrefix(cursor))
}

func (s *etcdExecutorStore) nextGroupName(ctx context.Context, prefix string, startKey string) (string, bool, error) {
	resp, err := s.client.Get(ctx, startKey,
		clientv3.WithRange(clientv3.GetPrefixRangeEnd(prefix)),
		clientv3.WithLimit(1),
		clientv3.WithKeysOnly(),
	)
	if err != nil {
		return "", false, err
	}
	if len(resp.Kvs) == 0 {
		return "", false, nil
	}
	name, ok := parseGroupName(string(resp.Kvs[0].Key))
	return name, ok, nil
}

func (s *etcdExecutorStore) listGroupInstances(ctx context.Context, name string) ([]serviceInstance, error) {
	resp, err := s.client.Get(ctx, executorGroupPrefix(name), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	instances := make([]serviceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si serviceInstance
		if err = json.Unmarshal(kv.Value, &si); err != nil {
			return nil, err
		}
		instances = append(instances, si)
	}
	return instances, nil
}

func executorIndexPrefix(name string) string {
	return fmt.Sprintf("%s/%s", indexer.ExecutorPrefix, name)
}

func executorGroupPrefix(name string) string {
	return fmt.Sprintf("%s/%s/", indexer.ExecutorPrefix, name)
}

func parseGroupName(key string) (string, bool) {
	prefix := executorIndexPrefix("")
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(key, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}
