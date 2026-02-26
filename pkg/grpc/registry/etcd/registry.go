package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const (
	// DefaultPrefix 默认服务注册前缀
	DefaultPrefix = "/grpc/services"
)

var typesMap = map[mvccpb.Event_EventType]registry.EventType{
	mvccpb.PUT:    registry.EventTypeAdd,
	mvccpb.DELETE: registry.EventTypeDelete,
}

type Registry struct {
	client *clientv3.Client
	prefix string // 服务注册前缀

	mutex       sync.RWMutex
	watchCancel []func()
	leases      map[string]func() // 控制生命周期
}

// NewRegistry 创建 Registry,使用默认前缀
func NewRegistry(c *clientv3.Client) (*Registry, error) {
	return NewRegistryWithPrefix(c, DefaultPrefix)
}

// NewRegistryWithPrefix 创建 Registry,使用自定义前缀
func NewRegistryWithPrefix(c *clientv3.Client, prefix string) (*Registry, error) {
	return &Registry{
		client: c,
		prefix: prefix,
		leases: make(map[string]func()),
	}, nil
}

func (r *Registry) Register(ctx context.Context, si registry.ServiceInstance) error {
	val, err := json.Marshal(si)
	if err != nil {
		return err
	}

	sess, err := concurrency.NewSession(r.client, concurrency.WithTTL(10))
	if err != nil {
		return err
	}

	_, err = r.client.Put(ctx, r.instanceKey(si), string(val), clientv3.WithLease(sess.Lease()))
	if err != nil {
		_ = sess.Close()
		return err
	}

	startCtx, cancel := context.WithCancel(context.Background())
	r.mutex.Lock()
	if r.leases == nil {
		r.leases = make(map[string]func())
	}
	key := r.instanceKey(si)
	if oldCancel, ok := r.leases[key]; ok {
		oldCancel()
	}
	r.leases[key] = cancel
	r.mutex.Unlock()

	go r.maintainLoop(startCtx, si, string(val), sess)
	return nil
}

func (r *Registry) maintainLoop(ctx context.Context, si registry.ServiceInstance, val string, sess *concurrency.Session) {
	for {
		select {
		case <-ctx.Done():
			_ = sess.Close()
			return
		case <-sess.Done():
			// session 失效（如网络断开），需要重新建立并注册
		}

		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			newSess, err := concurrency.NewSession(r.client, concurrency.WithTTL(10))
			if err != nil {
				time.Sleep(backoff)
				backoff = min(backoff*2, 30*time.Second)
				continue
			}

			putCtx, putCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err = r.client.Put(putCtx, r.instanceKey(si), val, clientv3.WithLease(newSess.Lease()))
			putCancel()

			if err != nil {
				_ = newSess.Close()
				time.Sleep(backoff)
				backoff = min(backoff*2, 30*time.Second)
				continue
			}

			// 重连并重新注册成功
			sess = newSess
			break
		}
	}
}

func (r *Registry) instanceKey(s registry.ServiceInstance) string {
	return fmt.Sprintf("%s/%s/%s", r.prefix, s.Name, s.Address)
}

func (r *Registry) UnRegister(ctx context.Context, si registry.ServiceInstance) error {
	r.mutex.Lock()
	if cancel, ok := r.leases[r.instanceKey(si)]; ok {
		cancel()
		delete(r.leases, r.instanceKey(si))
	}
	r.mutex.Unlock()

	_, err := r.client.Delete(ctx, r.instanceKey(si))
	return err
}

func (r *Registry) ListServices(ctx context.Context, name string) ([]registry.ServiceInstance, error) {
	resp, err := r.client.Get(ctx, r.serviceKey(name), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	res := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si registry.ServiceInstance
		err = json.Unmarshal(kv.Value, &si)
		if err != nil {
			return nil, err
		}
		res = append(res, si)
	}
	return res, nil
}

func (r *Registry) serviceKey(name string) string {
	return fmt.Sprintf("%s/%s", r.prefix, name)
}

func (r *Registry) Subscribe(name string) <-chan registry.Event {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = clientv3.WithRequireLeader(ctx)
	r.mutex.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mutex.Unlock()
	res := make(chan registry.Event)
	go func() {
		defer close(res)
		backoff := time.Second
		for {
			ch := r.client.Watch(ctx, r.serviceKey(name), clientv3.WithPrefix())
		WatchLoop:
			for {
				select {
				case <-ctx.Done():
					return
				case resp, ok := <-ch:
					if !ok || resp.Canceled {
						break WatchLoop
					}
					if resp.Err() != nil {
						continue
					}
					backoff = time.Second // 收到正常响应，重置退避时间
					for _, event := range resp.Events {
						res <- registry.Event{
							Type: typesMap[event.Type],
						}
					}
				}
			}
			// 断线重连前的退避
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
		}
	}()
	return res
}

func (r *Registry) Close() error {
	r.mutex.Lock()
	for _, cancel := range r.watchCancel {
		cancel()
	}
	for _, cancel := range r.leases {
		cancel()
	}
	r.leases = make(map[string]func())
	r.mutex.Unlock()
	// 因为 client 是外面传进来的，所以我们在上层进行控制，可能被其它的人使用着
	return nil
}
