package middleware

import (
	"net/http"
	"strconv"

	policyv1 "github.com/Duke1616/etask/api/proto/gen/ecmdb/policy/v1"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

const Resource = "TASK"

type CheckPolicyMiddlewareBuilder struct {
	policySvc policyv1.PolicyServiceClient
	logger    *elog.Component
	sp        session.Provider
}

func NewCheckPolicyMiddlewareBuilder(policySvc policyv1.PolicyServiceClient, sp session.Provider) *CheckPolicyMiddlewareBuilder {
	return &CheckPolicyMiddlewareBuilder{
		policySvc: policySvc,
		logger:    elog.DefaultLogger,
		sp:        sp,
	}
}

func (c *CheckPolicyMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		gCtx := &ginx.Context{Context: ctx}
		sess, err := c.sp.Get(gCtx)
		if err != nil {
			gCtx.AbortWithStatus(http.StatusForbidden)
			c.logger.Warn("用户未登录", elog.FieldErr(err))
			return
		}

		// 获取用户ID
		uid := sess.Claims().Uid
		resp, err := c.policySvc.Authorize(ctx.Request.Context(), &policyv1.AuthorizeReq{
			UserId:   strconv.FormatInt(uid, 10),
			Path:     ctx.Request.URL.Path,
			Method:   ctx.Request.Method,
			Resource: Resource,
		})

		if err != nil {
			gCtx.AbortWithStatus(http.StatusForbidden)
			c.logger.Error("调取权限接口失败", elog.FieldErr(err))
			return
		}

		if !resp.Allowed {
			gCtx.AbortWithStatus(http.StatusForbidden)
			c.logger.Warn("用户无权限",
				elog.Int64("uid", uid),
				elog.String("path", ctx.Request.URL.Path),
				elog.String("method", ctx.Request.Method),
				elog.String("reason", resp.Reason),
				elog.Any("roles", resp.Roles),
				elog.Any("matched_policies", resp.MatchedPolicies))
			return
		}

		c.logger.Debug("鉴权通过",
			elog.Int64("uid", uid),
			elog.Any("roles", resp.Roles))
	}
}
