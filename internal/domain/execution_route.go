package domain

import "fmt"

// ExecutionTransport 描述任务指令使用的传输通道。
type ExecutionTransport string

const (
	ExecutionTransportLocal ExecutionTransport = "LOCAL"
	ExecutionTransportHTTP  ExecutionTransport = "HTTP"
	ExecutionTransportGRPC  ExecutionTransport = "GRPC"
	ExecutionTransportMQ    ExecutionTransport = "MQ"
)

// String 返回传输通道的字符串值。
func (t ExecutionTransport) String() string {
	return string(t)
}

// ExecutionRoute 是创建执行记录时固定的派发路由。
// 重试和结果处理复用该快照，不再根据可能变化的资源池重新决定传输通道。
type ExecutionRoute struct {
	Transport    ExecutionTransport `json:"transport"`
	DispatchMode ExecMode           `json:"dispatch_mode"`
	PoolName     string             `json:"pool_name,omitempty"`
	TargetNodeID string             `json:"target_node_id,omitempty"`
	Topic        string             `json:"topic,omitempty"`
}

// Validate 校验路由内部约束。
func (r ExecutionRoute) Validate() error {
	switch r.Transport {
	case ExecutionTransportLocal, ExecutionTransportHTTP:
		if !r.DispatchMode.IsPush() {
			return fmt.Errorf("%s 执行路由只支持 PUSH", r.Transport)
		}
		return nil
	case ExecutionTransportGRPC:
		if r.PoolName == "" {
			return fmt.Errorf("gRPC 执行路由缺少资源池")
		}
		if !r.DispatchMode.IsPush() && !r.DispatchMode.IsPull() {
			return fmt.Errorf("gRPC 执行路由派发模式非法: %s", r.DispatchMode)
		}
		if r.DispatchMode.IsPush() && r.TargetNodeID == "" {
			return fmt.Errorf("gRPC PUSH 执行路由缺少目标节点")
		}
		return nil
	case ExecutionTransportMQ:
		if r.PoolName == "" {
			return fmt.Errorf("MQ 执行路由缺少资源池")
		}
		if r.Topic == "" {
			return fmt.Errorf("MQ 执行路由缺少 Topic")
		}
		if !r.DispatchMode.IsPush() {
			return fmt.Errorf("MQ 执行路由只支持 PUSH")
		}
		return nil
	default:
		return fmt.Errorf("执行路由传输通道非法: %s", r.Transport)
	}
}
