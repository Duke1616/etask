package domain

type ExecutorList struct {
	Executors  []Executor
	NextCursor string
}

type Executor struct {
	Name     string
	Desc     string
	Mode     string
	Handlers []ExecutorHandler
	Nodes    []ExecutorNode
}

type ExecutorNode struct {
	ID      string
	Address string
}

type ExecutorHandler struct {
	Name     string
	Desc     string
	Metadata []ExecutorParameter
}

type ExecutorParameter struct {
	Key      string
	Desc     string
	Secret   bool
	Required bool
	Bindings map[string]ExecutorBinding
	Default  string
}

type ExecutorBinding struct {
	Label       string
	Placeholder string
	Component   string
	Config      map[string]string
}
