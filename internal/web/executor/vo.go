package executor

type Executor struct {
	Name     string          `json:"name"`     // 分组的执行器服务名
	Desc     string          `json:"desc"`     // 执行器的总体功能描述
	Mode     string          `json:"mode"`     // 执行器模式
	Handlers []HandlerDetail `json:"handlers"` // 该分组下所有节点共同支持的处理方法
	Nodes    []NodeDetail    `json:"nodes"`    // 该服务名下的所有在线节点
}

type NodeDetail struct {
	ID      string `json:"id"`      // 节点唯一ID
	Address string `json:"address"` // 节点网络地址
}

// HandlerDetail 处理器详情
// NOTE: 这里的 Metadata 使用 ParameterVO 而非 executor.Parameter 是为了避免
// 接口类型 (Binding) 在 JSON 反序列化时丢失具体类型信息，从而方便前端使用。
type HandlerDetail struct {
	Name     string        `json:"name"`
	Desc     string        `json:"desc"`
	Metadata []ParameterVO `json:"metadata"`
}

type ParameterVO struct {
	Key      string               `json:"key"`
	Desc     string               `json:"desc"`
	Secret   bool                 `json:"secret"` // 是否是加密参数
	Required bool                 `json:"required"`
	Bindings map[string]BindingVO `json:"bindings"` // 支持的绑定能力映射
	Default  string               `json:"default"`  // 默认值
}

type BindingVO struct {
	Label       string            `json:"label"`       // 展示给用户的选项名称
	Placeholder string            `json:"placeholder"` // 占位符
	Component   string            `json:"component"`   // UI 渲染控件提示
	Config      map[string]string `json:"config"`      // 扩展配置提示
}
