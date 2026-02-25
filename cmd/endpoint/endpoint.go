package endpoint

import (
	"context"
	"fmt"
	"strings"
	"time"

	endpointv1 "github.com/Duke1616/ework-runner/api/proto/gen/ecmdb/endpoint/v1"
	"github.com/Duke1616/ework-runner/cmd/scheduler/ioc"
	"github.com/Duke1616/ework-runner/pkg/ginx/middleware"
	"github.com/gotomicro/ego/server/egin"
	"github.com/spf13/cobra"
)

const Resource = middleware.Resource

var Cmd = &cobra.Command{
	Use:   "endpoint",
	Short: "ework-runner endpoint",
	Long:  "注册所有路由信息到 Endpoint 中，用于动态菜单API鉴权中使用",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := ioc.InitSchedulerApp()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := initEndpoint(ctx, app.Web, app.EndpointSvc)
		if err != nil {
			panic(err)
		}
		fmt.Println("端点初始化完成")
		return nil
	},
}

// 生成端点路由信息、方便菜单权限绑定路由
func initEndpoint(ctx context.Context, web *egin.Component, endpointSvc endpointv1.EndpointServiceClient) error {
	routes := web.Routes()
	var endpoints []*endpointv1.Endpoint

	fmt.Printf("扫描到 %d 个路由\n", len(routes))

	// 分析每个路由的权限要求
	for _, route := range routes {
		// 基于 Handler 名称分析权限要求
		status := analyzeRoutePermissionsByHandler(route.Handler)

		// 创建端点信息
		ep := &endpointv1.Endpoint{
			Path:         route.Path,
			Method:       route.Method,
			Resource:     Resource,
			IsAuth:       status.IsAuth,
			IsAudit:      status.IsAudit,
			IsPermission: status.IsPermission,
		}

		endpoints = append(endpoints, ep)

		fmt.Printf("路由: %-6s %-30s - 登录:%t 审计:%t 权限:%t\n",
			route.Method, route.Path, status.IsAuth, status.IsAudit, status.IsPermission)
	}

	count, err := endpointSvc.BatchRegister(ctx, &endpointv1.BatchRegisterEndpointsReq{
		Resource:  Resource,
		Endpoints: endpoints,
	})
	if err != nil {
		return fmt.Errorf("注册资源 %s 的端点失败: %w", Resource, err)
	}
	fmt.Printf("资源 %s: 注册了 %s 个端点\n", Resource, count)
	return nil
}

// RouteMiddlewareStatus 路由中间件状态
type RouteMiddlewareStatus struct {
	IsAuth       bool // 是否需要登录
	IsAudit      bool // 是否需要审计
	IsPermission bool // 是否需要权限验证
}

// 基于 Handler 名称分析权限要求
func analyzeRoutePermissionsByHandler(handler string) RouteMiddlewareStatus {
	status := RouteMiddlewareStatus{}

	// 基于 Handler 名称判断权限要求
	if strings.Contains(handler, "PrivateRoutes") {
		// PrivateRoutes 需要完整权限
		status.IsAuth = true
		status.IsAudit = false
		status.IsPermission = true
	} else if strings.Contains(handler, "PublicRoutes") {
		// PublicRoutes 是公开路由
		status.IsAuth = false
		status.IsAudit = false
		status.IsPermission = false
	} else {
		// 其他情况默认为需要完整权限
		status.IsAuth = true
		status.IsAudit = true
		status.IsPermission = true
	}

	return status
}
