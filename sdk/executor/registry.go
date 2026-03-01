package executor

// HandlerRegistry 处理器注册中心，由于 Executor 和 Agent Service 共用
type HandlerRegistry struct {
	handlers map[string]TaskHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]TaskHandler),
	}
}

// Register 注册一个或多个处理器
func (r *HandlerRegistry) Register(handlers ...TaskHandler) {
	for _, h := range handlers {
		r.handlers[h.Name()] = h
	}
}

// Get 根据名称获取处理器
func (r *HandlerRegistry) Get(name string) (TaskHandler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}

// ListMetas 返回所有处理器的元数据清单 (用于上报、展示)
func (r *HandlerRegistry) ListMetas() []HandlerMeta {
	metas := make([]HandlerMeta, 0, len(r.handlers))
	for _, h := range r.handlers {
		metas = append(metas, HandlerMeta{
			Name: h.Name(),
			Desc: h.Desc(),
		})
	}
	return metas
}

// Handlers 获取原始 map
func (r *HandlerRegistry) Handlers() map[string]TaskHandler {
	return r.handlers
}
