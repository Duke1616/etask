package grpc

import (
	"context"

	codebookv1 "github.com/Duke1616/etask/api/proto/gen/etask/codebook/v1"
	"github.com/Duke1616/etask/internal/domain"
	codebookSvc "github.com/Duke1616/etask/internal/service/codebook"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CodebookServer 对外提供脚本模板查询能力。
type CodebookServer struct {
	codebookv1.UnimplementedCodebookServiceServer
	svc    codebookSvc.Service
	logger *elog.Component
}

// NewCodebookServer 创建脚本模板 gRPC 服务端。
func NewCodebookServer(svc codebookSvc.Service) *CodebookServer {
	return &CodebookServer{
		svc:    svc,
		logger: elog.DefaultLogger.With(elog.FieldComponentName("grpc.CodebookServer")),
	}
}

// GetCodebookByIdentifier 根据业务唯一标识获取脚本模板。
func (s *CodebookServer) GetCodebookByIdentifier(ctx context.Context, req *codebookv1.GetCodebookByIdentifierRequest) (*codebookv1.GetCodebookByIdentifierResponse, error) {
	c, err := s.svc.GetByIdentifier(ctx, req.GetIdentifier())
	if err != nil {
		s.logger.Error("获取脚本模板失败", elog.String("identifier", req.GetIdentifier()), elog.FieldErr(err))
		return nil, status.Errorf(codes.NotFound, "codebook not found: %v", err)
	}
	return &codebookv1.GetCodebookByIdentifierResponse{Codebook: s.toProto(c)}, nil
}

func (s *CodebookServer) toProto(c domain.Codebook) *codebookv1.Codebook {
	return &codebookv1.Codebook{
		Id:         c.ID,
		Name:       c.Name,
		Owner:      c.Owner,
		Code:       c.Code,
		Language:   c.Language,
		Secret:     c.Secret,
		Identifier: c.Identifier,
		Ctime:      c.CTime,
		Utime:      c.UTime,
	}
}
