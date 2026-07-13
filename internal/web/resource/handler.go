package resource

import (
	"encoding/json"

	"github.com/Duke1616/eiam/pkg/web/capability"
	"github.com/Duke1616/etask/internal/domain"
	poolSvc "github.com/Duke1616/etask/internal/service/pool"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	catalogSvc poolSvc.CatalogService
	capability.IRegistry
}

func NewHandler(catalogSvc poolSvc.CatalogService) *Handler {
	return &Handler{
		catalogSvc: catalogSvc,
		IRegistry:  capability.NewRegistry("task", "resource", "执行资源"),
	}
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {
}

func (h *Handler) IdentifyRoutes(_ *gin.Engine) {
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/api/resource")

	g.GET("/list", h.Capability("执行资源列表", "view").
		Handle(ginx.B[ListReq](h.List)),
	)
}

func (h *Handler) List(ctx *ginx.Context, req ListReq) (ginx.Result, error) {
	page, err := h.catalogSvc.ListAuthorizedPools(ctx, poolSvc.CatalogListRequest{
		Kind:    domain.ExecutionPoolKind(req.Kind),
		Offset:  req.Offset,
		Limit:   req.Limit,
		Keyword: req.Keyword,
	})
	if err != nil {
		return systemErrorResult, err
	}

	matcher := poolSvc.NewBindingMatcher(page.Bindings)
	resources := lo.FilterMap(page.Pools, func(pool domain.ExecutionPool, _ int) (ResourceVO, bool) {
		resource := h.toVO(pool)
		if len(resource.Handlers) == 0 {
			return resource, matcher.Allow(pool.Name, "")
		}
		resource.Handlers = lo.Filter(resource.Handlers, func(handler HandlerDetail, _ int) bool {
			return matcher.Allow(pool.Name, handler.Name)
		})
		return resource, len(resource.Handlers) > 0
	})

	return ginx.Result{
		Data: ListResp{
			Resources: resources,
			Total:     page.Total,
		},
		Msg: "success",
	}, nil
}

func (h *Handler) toVO(pool domain.ExecutionPool) ResourceVO {
	return ResourceVO{
		Name:     pool.Name,
		Desc:     pool.Desc,
		Kind:     pool.Kind.String(),
		Mode:     pool.Mode.String(),
		Topic:    pool.Metadata["topic"],
		Handlers: parseHandlers(pool.Metadata["supported_handlers"]),
	}
}

func parseHandlers(raw string) []HandlerDetail {
	if raw == "" {
		return nil
	}
	var handlers []HandlerDetail
	_ = json.Unmarshal([]byte(raw), &handlers)
	return handlers
}
