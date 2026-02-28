package ioc

import (
	"strings"
	"time"

	"github.com/Duke1616/ework-runner/internal/web/executor"
	"github.com/Duke1616/ework-runner/internal/web/task"
	"github.com/Duke1616/ework-runner/pkg/ginx/middleware"
	"github.com/ecodeclub/ginx/session"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/server/egin"
)

func InitGinWebServer(mdls []gin.HandlerFunc, checkPolicyMiddleware *middleware.CheckPolicyMiddlewareBuilder,
	sp session.Provider, taskHdl *task.Handler, executorHdl *executor.Handler) *egin.Component {
	session.SetDefaultProvider(sp)

	server := egin.DefaultContainer().Build(egin.WithPort(8765))
	server.Use(mdls...)

	// 注册公开路由
	taskHdl.PublicRoutes(server.Engine)
	executorHdl.PublicRoutes(server.Engine)

	// 验证是否登录
	server.Use(session.CheckLoginMiddleware())

	// 检查权限策略
	server.Use(checkPolicyMiddleware.Build())

	// 注册私有路由
	taskHdl.PrivateRoutes(server.Engine)
	executorHdl.PrivateRoutes(server.Engine)

	return server
}

func InitGinMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowHeaders:     []string{"Content-Type", "Authorization"},
			ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost") {
					return true
				}
				return strings.Contains(origin, "your_domain.com")
			},
			MaxAge: 12 * time.Hour,
		}),
	}
}
