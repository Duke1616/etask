package ioc

import (
	"net"
	"strings"
	"time"

	"github.com/Duke1616/ecmdb/pkg/policy"
	"github.com/Duke1616/etask/internal/agent/web"
	"github.com/Duke1616/etask/internal/web/executor"
	"github.com/Duke1616/etask/internal/web/task"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
)

const Resource = "TASK"

func InitGinWebServer(mdls []gin.HandlerFunc, sdk *policy.SDK,
	taskHdl *task.Handler, executorHdl *executor.Handler, agentHdl *web.Handler, listener net.Listener) *egin.Component {

	server := egin.Load("server.egin").Build(egin.WithListener(listener))
	server.Use(mdls...)

	// 注册公开路由
	taskHdl.PublicRoutes(server.Engine)
	executorHdl.PublicRoutes(server.Engine)
	agentHdl.PublicRoutes(server.Engine)

	// 登录检查
	server.Use(sdk.CheckLogin())

	// 权限策略检查
	server.Use(sdk.CheckPolicy(Resource))

	// 注册私有路由
	taskHdl.PrivateRoutes(server.Engine)
	executorHdl.PrivateRoutes(server.Engine)
	agentHdl.PrivateRoutes(server.Engine)

	return server
}

func InitGinMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		corsHdl(),
		accessLogger(),
		func(ctx *gin.Context) {
		},
	}
}

func corsHdl() gin.HandlerFunc {
	return cors.New(cors.Config{
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
	})
}

// accessLogger 自定义 access 日志中间件
func accessLogger() gin.HandlerFunc {
	// 关闭默认的日志输出
	econf.Set("server.egin.enableAccessInterceptor", false)

	// NOTE: ego DefaultLogger 针对框架内部做了 caller skip 校准，直接从用户代码调用需减一层
	logger := elog.DefaultLogger.With(elog.FieldComponentName("access")).WithCallerSkip(-1)
	return func(ctx *gin.Context) {
		beg := time.Now()
		ctx.Next()
		cost := time.Since(beg)

		fields := []elog.Field{
			elog.FieldMethod(ctx.Request.Method + "." + ctx.FullPath()),
			elog.FieldAddr(ctx.Request.URL.RequestURI()),
			elog.FieldCost(cost),
			elog.FieldCode(int32(ctx.Writer.Status())),
		}

		logger.Info("access", fields...)
	}
}
