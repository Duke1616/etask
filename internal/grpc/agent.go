package grpc

import (
	"context"

	executorv1 "github.com/Duke1616/ework-runner/api/proto/gen/etask/executor/v1"
	"github.com/Duke1616/ework-runner/internal/repository"
)

// AgentServer 实现调度中心的 Agent 拉取服务
type AgentServer struct {
	executorv1.UnimplementedAgentServiceServer
	execRepo repository.TaskExecutionRepository
}

func NewAgentServer(execRepo repository.TaskExecutionRepository) *AgentServer {
	return &AgentServer{
		execRepo: execRepo,
	}
}

// PullTask 响应执行节点的拉取请求
func (s *AgentServer) PullTask(ctx context.Context, req *executorv1.PullTaskRequest) (*executorv1.PullTaskResponse, error) {
	serviceName := req.GetServiceName()
	nodeId := req.GetNodeId()
	if serviceName == "" {
		return &executorv1.PullTaskResponse{HasTask: false}, nil
	}
	if nodeId == "" {
		nodeId = serviceName // 向后兼容
	}

	// 1. 在数据库中寻找并乐观抢占一条状态为 WAITING_PULL 且 Service(Group) 匹配的执行记录
	// 这里将 nodeId 真实落库记录为 executor_node_id
	exec, err := s.execRepo.ClaimPullTask(ctx, serviceName, nodeId)
	if err != nil {
		// 没有活儿，或者没抢到
		return &executorv1.PullTaskResponse{HasTask: false}, nil
	}

	handlerName := ""
	if exec.Task.GrpcConfig != nil {
		handlerName = exec.Task.GrpcConfig.HandlerName
	}

	// 2. 抢占成功，直接将执行指令下发
	return &executorv1.PullTaskResponse{
		HasTask: true,
		TaskReq: &executorv1.ExecuteRequest{
			Eid:             exec.ID,
			TaskId:          exec.Task.ID,
			TaskName:        exec.Task.Name,
			TaskHandlerName: handlerName,
			Params:          exec.GRPCParams(),
		},
	}, nil
}
