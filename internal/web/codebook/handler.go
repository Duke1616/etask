package codebook

import (
	"errors"

	"github.com/Duke1616/eiam/pkg/web/capability"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/errs"
	codebookSvc "github.com/Duke1616/etask/internal/service/codebook"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc codebookSvc.Service
	capability.IRegistry
}

func NewHandler(svc codebookSvc.Service) *Handler {
	return &Handler{
		svc:       svc,
		IRegistry: capability.NewRegistry("task", "codebook", "脚本引擎"),
	}
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/codebook")
	g.POST("/create", h.Capability("创建脚本模板", "add").
		Handle(ginx.B[CreateReq](h.Create)),
	)
	g.POST("/list", h.Capability("脚本模板列表", "view").
		Handle(ginx.B[ListReq](h.List)),
	)
	g.GET("/detail/:id", h.Capability("脚本模板详情", "get").
		Handle(ginx.W(h.Detail)),
	)
	g.POST("/update", h.Capability("更新脚本模板", "edit").
		Handle(ginx.B[UpdateReq](h.Update)),
	)
	g.DELETE("/delete/:id", h.Capability("删除脚本模板", "delete").
		Handle(ginx.W(h.Delete)),
	)
}

func (h *Handler) IdentifyRoutes(_ *gin.Engine) {
}

func (h *Handler) Create(ctx *ginx.Context, req CreateReq) (ginx.Result, error) {
	id, err := h.svc.Create(ctx, h.toDomain(req))
	if err != nil {
		return h.translateError(err), err
	}
	return ginx.Result{Data: id, Msg: "success"}, nil
}

func (h *Handler) Detail(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return invalidCodebookIDError, err
	}
	c, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return h.translateError(err), err
	}
	return ginx.Result{Data: h.toVO(c), Msg: "success"}, nil
}

func (h *Handler) List(ctx *ginx.Context, req ListReq) (ginx.Result, error) {
	cs, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "success",
		Data: ListCodebooksResp{
			Total: total,
			Codebooks: slice.Map(cs, func(_ int, src domain.Codebook) CodebookVO {
				return h.toVO(src)
			}),
		},
	}, nil
}

func (h *Handler) Update(ctx *ginx.Context, req UpdateReq) (ginx.Result, error) {
	count, err := h.svc.Update(ctx, h.toUpdateDomain(req))
	if err != nil {
		return h.translateError(err), err
	}
	return ginx.Result{Data: count, Msg: "success"}, nil
}

func (h *Handler) Delete(ctx *ginx.Context) (ginx.Result, error) {
	id, err := ctx.Param("id").AsInt64()
	if err != nil {
		return invalidCodebookIDError, err
	}
	count, err := h.svc.Delete(ctx, id)
	if err != nil {
		return h.translateError(err), err
	}
	return ginx.Result{Data: count, Msg: "success"}, nil
}

func (h *Handler) translateError(err error) ginx.Result {
	if errors.Is(err, errs.ErrInvalidParameter) {
		return invalidParameterResult(err)
	}
	return systemErrorResult
}

func (h *Handler) toDomain(req CreateReq) domain.Codebook {
	return domain.Codebook{
		Name:       req.Name,
		Owner:      req.Owner,
		Code:       req.Code,
		Language:   req.Language,
		Identifier: req.Identifier,
	}
}

func (h *Handler) toUpdateDomain(req UpdateReq) domain.Codebook {
	return domain.Codebook{
		ID:       req.ID,
		Name:     req.Name,
		Owner:    req.Owner,
		Code:     req.Code,
		Language: req.Language,
	}
}

func (h *Handler) toVO(req domain.Codebook) CodebookVO {
	return CodebookVO{
		ID:         req.ID,
		Name:       req.Name,
		Owner:      req.Owner,
		Identifier: req.Identifier,
		Code:       req.Code,
		Language:   req.Language,
		Secret:     req.Secret,
		CTime:      req.CTime,
		UTime:      req.UTime,
	}
}
