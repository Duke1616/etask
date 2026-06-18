package grpc

import (
	"context"

	runnerv1 "github.com/Duke1616/etask/api/proto/gen/etask/runner/v1"
	"github.com/Duke1616/etask/internal/domain"
	runnerSvc "github.com/Duke1616/etask/internal/service/runner"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RunnerServer 对外提供执行单元查询能力。
type RunnerServer struct {
	runnerv1.UnimplementedRunnerServiceServer
	svc    runnerSvc.Service
	logger *elog.Component
}

// NewRunnerServer 创建执行单元 gRPC 服务端。
func NewRunnerServer(svc runnerSvc.Service) *RunnerServer {
	return &RunnerServer{
		svc:    svc,
		logger: elog.DefaultLogger.With(elog.FieldComponentName("grpc.RunnerServer")),
	}
}

// FindRunnerByCodebookUidAndTag 根据脚本模板 UID 和派发标签获取执行单元。
func (s *RunnerServer) FindRunnerByCodebookUidAndTag(ctx context.Context, req *runnerv1.FindRunnerByCodebookUidAndTagRequest) (*runnerv1.FindRunnerByCodebookUidAndTagResponse, error) {
	r, err := s.svc.FindByCodebookUIDAndTag(ctx, req.GetCodebookUid(), req.GetTag())
	if err != nil {
		s.logger.Error("获取执行单元失败",
			elog.String("codebookUID", req.GetCodebookUid()),
			elog.String("tag", req.GetTag()),
			elog.FieldErr(err))
		return nil, status.Errorf(codes.NotFound, "runner not found: %v", err)
	}
	return &runnerv1.FindRunnerByCodebookUidAndTagResponse{Runner: s.toProto(r)}, nil
}

func (s *RunnerServer) toProto(r domain.Runner) *runnerv1.Runner {
	return &runnerv1.Runner{
		Id:             r.ID,
		Name:           r.Name,
		CodebookUid:    r.CodebookUID,
		CodebookSecret: r.CodebookSecret,
		Kind:           r.Kind.String(),
		Target:         r.Target,
		Handler:        r.Handler,
		Tags:           r.Tags,
		Action:         uint32(r.Action),
		Desc:           r.Desc,
		Ctime:          r.CTime,
		Utime:          r.UTime,
		Variables: slice.Map(r.Variables, func(_ int, src domain.RunnerVariable) *runnerv1.Variable {
			return &runnerv1.Variable{
				Key:    src.Key,
				Value:  src.Value,
				Secret: src.Secret,
			}
		}),
	}
}
