package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Duke1616/ework-runner/pkg/grpc/registry"
	"github.com/gotomicro/ego/core/elog"
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
	logger      *elog.Component
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
		logger: elog.DefaultLogger.With(elog.FieldComponentName("grpc.registry.etcd")),
	}, nil
}

func (r *Registry) Register(ctx context.Context, si registry.ServiceInstance) error {
	val, err := json.Marshal(si)
	if err != nil {
		return err
	}

	// NOTE: 工业级标准建议 TTL >= 30s 以防御瞬时网络波动
	sess, err := concurrency.NewSession(r.client, concurrency.WithTTL(30))
	if err != nil {
		return err
	}

	key := r.instanceKey(si)
	_, err = r.client.Put(ctx, key, string(val), clientv3.WithLease(sess.Lease()))
	if err != nil {
		_ = sess.Close()
		return err
	}

	startCtx, cancel := context.WithCancel(context.Background())
	r.mutex.Lock()
	if r.leases == nil {
		r.leases = make(map[string]func())
	}
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
			r.logger.Warn("Session 失效，准备重新注册", elog.String("instance", si.Address))
		}

		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 重新建立 Session 和 Lease
			newSess, err := concurrency.NewSession(r.client, concurrency.WithTTL(30))
			if err != nil {
				r.logger.Error("创建 Session 失败", elog.FieldErr(err), elog.String("backoff", backoff.String()))
				time.Sleep(backoff)
				backoff = min(backoff*2, 30*time.Second)
				continue
			}

			putCtx, putCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err = r.client.Put(putCtx, r.instanceKey(si), val, clientv3.WithLease(newSess.Lease()))
			putCancel()

			if err != nil {
				r.logger.Error("重新注册实例失败", elog.FieldErr(err), elog.String("backoff", backoff.String()))
				_ = newSess.Close()
				time.Sleep(backoff)
				backoff = min(backoff*2, 30*time.Second)
				continue
			}

			r.logger.Info("重新注册实例成功", elog.String("instance", si.Address))
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
	key := r.serviceKey(name)
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	r.logger.Info("ListServices查询结果", elog.String("service", name), elog.Int("count", len(resp.Kvs)))
	res := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si registry.ServiceInstance
		err = json.Unmarshal(kv.Value, &si)
		if err != nil {
			r.logger.Warn("解析实例数据失败", elog.String("key", string(kv.Key)), elog.FieldErr(err))
			continue
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
	r.mutex.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mutex.Unlock()

	res := make(chan registry.Event, 16)
	go func() {
		defer close(res)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 每次重启 Watch，为了防止漏事件，强行发一次 EventTypeAdd 触发全量同步
			select {
			case res <- registry.Event{Type: registry.EventTypeAdd}:
			case <-ctx.Done():
				return
			}

			// 监听目标服务的前缀，不需要预先获取 revision，由 etcd 客户端内部接管
			ch := r.client.Watch(ctx, r.serviceKey(name), clientv3.WithPrefix())
		WatchLoop:
			for {
				select {
				case <-ctx.Done():
					return
				case resp, ok := <-ch:
					if !ok || resp.Canceled || resp.Err() != nil {
						// 只要断联/出错（如 Compacted），直接退出当前 Watch 循环，延时后重新发起全新的 Watch
						time.Sleep(time.Second)
						break WatchLoop
					}

					for _, event := range resp.Events {
						select {
						case res <- registry.Event{Type: typesMap[event.Type]}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
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
