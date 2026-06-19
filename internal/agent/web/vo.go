package web

type ListAgentsReq struct {
	Limit   int64  `form:"limit"`
	Cursor  string `form:"cursor"`
	Keyword string `form:"keyword"`
}

type ListAgentsResp struct {
	Agents     []Agent `json:"agents"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type Agent struct {
	Name     string          `json:"name"`     // 分组的代理名称
	Desc     string          `json:"desc"`     // 代理的总体功能描述
	Topic    string          `json:"topic"`    // 接收队列
	Handlers []HandlerDetail `json:"handlers"` // 该分组下所有节点共同支持的处理方法
	Nodes    []NodeDetail    `json:"nodes"`    // 该服务名下的所有在线节点
}

type NodeDetail struct {
	ID      string `json:"id"`      // 节点唯一ID
	Address string `json:"address"` // 节点网络地址
}

type HandlerDetail struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}
