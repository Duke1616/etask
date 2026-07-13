package domain

import "strings"

const ExecutionPoolHandlerWildcard = "*"

// ExecutionPoolKind 描述执行资源池承载的执行资源类型。
type ExecutionPoolKind string

const (
	// ExecutionPoolKindExecutor 表示通过 gRPC executor 节点执行。
	ExecutionPoolKindExecutor ExecutionPoolKind = "EXECUTOR"
	// ExecutionPoolKindAgent 表示通过 Agent 通道执行。
	ExecutionPoolKindAgent ExecutionPoolKind = "AGENT"
)

// String 返回执行资源类型的字符串值。
func (k ExecutionPoolKind) String() string {
	return string(k)
}

// ExecutionPoolMode 描述执行资源池的派发模式。
type ExecutionPoolMode string

const (
	// ExecutionPoolModePush 表示调度中心主动推送任务。
	ExecutionPoolModePush ExecutionPoolMode = "PUSH"
	// ExecutionPoolModePull 表示执行端主动拉取任务。
	ExecutionPoolModePull ExecutionPoolMode = "PULL"
	// ExecutionPoolModeMQ 表示通过消息队列派发任务。
	ExecutionPoolModeMQ ExecutionPoolMode = "MQ"
)

// String 返回派发模式的字符串值。
func (m ExecutionPoolMode) String() string {
	return string(m)
}

// ExecutionPoolIsolation 描述执行资源池的租户隔离级别。
type ExecutionPoolIsolation string

const (
	// ExecutionPoolIsolationShared 表示资源池可被多个租户共享。
	ExecutionPoolIsolationShared ExecutionPoolIsolation = "SHARED"
	// ExecutionPoolIsolationDedicated 表示资源池仅服务专属租户。
	ExecutionPoolIsolationDedicated ExecutionPoolIsolation = "DEDICATED"
)

// String 返回隔离级别的字符串值。
func (i ExecutionPoolIsolation) String() string {
	return string(i)
}

// ExecutionPoolStatus 描述执行资源池状态。
type ExecutionPoolStatus string

const (
	// ExecutionPoolStatusEnabled 表示资源池可用。
	ExecutionPoolStatusEnabled ExecutionPoolStatus = "ENABLED"
	// ExecutionPoolStatusDisabled 表示资源池不可用。
	ExecutionPoolStatusDisabled ExecutionPoolStatus = "DISABLED"
)

// String 返回资源池状态的字符串值。
func (s ExecutionPoolStatus) String() string {
	return string(s)
}

// ExecutionPoolBindingStatus 描述租户与执行资源池绑定关系状态。
type ExecutionPoolBindingStatus string

const (
	// ExecutionPoolBindingStatusEnabled 表示绑定可用。
	ExecutionPoolBindingStatusEnabled ExecutionPoolBindingStatus = "ENABLED"
	// ExecutionPoolBindingStatusDisabled 表示绑定不可用。
	ExecutionPoolBindingStatusDisabled ExecutionPoolBindingStatus = "DISABLED"
)

// String 返回绑定状态的字符串值。
func (s ExecutionPoolBindingStatus) String() string {
	return string(s)
}

// ExecutionPool 描述可被调度系统使用的一组执行资源。
type ExecutionPool struct {
	ID             int64
	Name           string
	Kind           ExecutionPoolKind
	Mode           ExecutionPoolMode
	IsolationLevel ExecutionPoolIsolation
	Desc           string
	Status         ExecutionPoolStatus
	Metadata       map[string]string
	CTime          int64
	UTime          int64
}

// ExecutionPoolBinding 描述租户对执行资源池及可选 handler 的授权关系。
type ExecutionPoolBinding struct {
	ID          int64
	TenantID    int64
	PoolName    string
	HandlerName string
	Status      ExecutionPoolBindingStatus
	Desc        string
	CTime       int64
	UTime       int64
}

// NormalizeExecutionPoolHandlerName 归一化 handler 授权维度。
// 空字符串和 * 都表示整个资源池。
func NormalizeExecutionPoolHandlerName(handlerName string) string {
	handlerName = strings.TrimSpace(handlerName)
	if handlerName == ExecutionPoolHandlerWildcard {
		return ""
	}
	return handlerName
}

// IsWildcard 判断当前绑定是否授权整个资源池。
func (b ExecutionPoolBinding) IsWildcard() bool {
	return NormalizeExecutionPoolHandlerName(b.HandlerName) == ""
}
