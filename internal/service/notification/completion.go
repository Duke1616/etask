package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"

	notificationv1 "github.com/Duke1616/etask/api/proto/gen/ealert/notification/v1"
	templatev1 "github.com/Duke1616/etask/api/proto/gen/ealert/template/v1"
	"github.com/Duke1616/etask/internal/domain"
)

const maxConcurrentDispatch = 10

// CompletionNotifier 负责投递任务执行终态通知与业务结果消息。
type CompletionNotifier interface {
	// Notify 根据已命中的系统通知规则投递执行终态快照。
	Notify(ctx context.Context, rule domain.ExecutionNotificationRule, execution domain.TaskExecution) error
	// DispatchBusinessNotifications 根据任务的业务通知配置投递产物中的 notifications 列表。
	DispatchBusinessNotifications(ctx context.Context, config domain.BusinessNotificationConfig, execution domain.TaskExecution) error
}

type alertCompletionNotifier struct {
	notificationClient notificationv1.NotificationServiceClient
	templateClient     templatev1.TemplateServiceClient
}

// NewEAlertCompletionNotifier 创建基于 EAlert 的任务执行通知器。
func NewEAlertCompletionNotifier(notificationClient notificationv1.NotificationServiceClient,
	templateClient templatev1.TemplateServiceClient) CompletionNotifier {
	return &alertCompletionNotifier{
		notificationClient: notificationClient,
		templateClient:     templateClient,
	}
}

// Notify 构造任务执行终态通知请求（如运维大盘/报警），调用 EAlert 并校验每个渠道的投递结果。
func (n *alertCompletionNotifier) Notify(ctx context.Context, rule domain.ExecutionNotificationRule,
	execution domain.TaskExecution) error {
	if n == nil || n.notificationClient == nil {
		return fmt.Errorf("EAlert 通知服务客户端不能为空")
	}
	if err := validateExecutionSnapshot(execution); err != nil {
		return err
	}
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("任务执行通知规则无效: %w", err)
	}

	templateSetID := rule.TemplateSetID
	if templateSetID == 0 {
		var err error
		templateSetID, err = n.resolveBuiltinTemplateSetID(ctx, rule)
		if err != nil {
			return err
		}
	}

	request, err := buildDispatchNotificationRequest(rule, execution, templateSetID)
	if err != nil {
		return err
	}

	response, err := n.notificationClient.DispatchNotification(ctx, request)
	if err != nil {
		return fmt.Errorf("调用 EAlert 投递任务通知失败: %w", err)
	}

	return validateDispatchNotificationResponse(response)
}

// DispatchBusinessNotifications 根据任务配置的业务通知规则，并发提取产物中的 notifications 列表并逐一投递。
func (n *alertCompletionNotifier) DispatchBusinessNotifications(ctx context.Context,
	config domain.BusinessNotificationConfig,
	execution domain.TaskExecution) error {
	if n == nil || n.notificationClient == nil {
		return fmt.Errorf("EAlert 通知服务客户端不能为空")
	}
	if !config.Enabled {
		return nil
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("业务通知配置无效: %w", err)
	}

	payload, _ := parseTaskResultPayload(execution.TaskResult)

	switch config.Mode {
	case domain.BusinessRecipientModeStatic:
		// 静态指定模式：任务结果不为空即可向预设人员发送汇总通知（支持仅输出汇总指标而无明细列表）
		if execution.TaskResult == "" {
			return nil
		}
		return n.dispatchStaticBusinessNotification(ctx, config, execution, payload)

	case domain.BusinessRecipientModeDynamic:
		// 动态识别模式：必须包含非空的 notifications 待投递条目
		if payload == nil || len(payload.Notifications) == 0 {
			return nil
		}
		return n.dispatchBusinessNotificationItems(ctx, config, execution, payload.Notifications)

	default:
		return nil
	}
}

func (n *alertCompletionNotifier) dispatchStaticBusinessNotification(ctx context.Context,
	config domain.BusinessNotificationConfig, execution domain.TaskExecution, payload *TaskResultPayload) error {
	req, err := buildStaticBusinessNotificationRequest(config, execution, payload)
	if err != nil {
		return err
	}
	return n.dispatchSingleBusinessNotification(ctx, req, "static-recipients")
}

func (n *alertCompletionNotifier) dispatchBusinessNotificationItems(ctx context.Context,
	config domain.BusinessNotificationConfig, execution domain.TaskExecution, items []BusinessNotificationItem) error {
	sem := make(chan struct{}, maxConcurrentDispatch)
	var (
		wg           sync.WaitGroup
		errMu        sync.Mutex
		dispatchErrs error
	)

	appendErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		dispatchErrs = errors.Join(dispatchErrs, err)
		errMu.Unlock()
	}

dispatchLoop:
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			appendErr(err)
			break dispatchLoop
		}

		req, err := buildBusinessNotificationRequest(config, execution, i, item)
		if err != nil {
			appendErr(err)
			continue
		}

		select {
		case <-ctx.Done():
			appendErr(ctx.Err())
			break dispatchLoop
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(r *notificationv1.DispatchNotificationRequest, target string) {
			defer func() {
				<-sem
				wg.Done()
			}()
			appendErr(n.dispatchSingleBusinessNotification(ctx, r, target))
		}(req, item.Target)
	}

	wg.Wait()
	return dispatchErrs
}

func (n *alertCompletionNotifier) dispatchSingleBusinessNotification(ctx context.Context,
	req *notificationv1.DispatchNotificationRequest, target string) error {
	resp, err := n.notificationClient.DispatchNotification(ctx, req)
	if err != nil {
		return fmt.Errorf("业务通知投递目标[%s]失败: %w", target, err)
	}
	if err = validateDispatchNotificationResponse(resp); err != nil {
		return fmt.Errorf("业务通知投递目标[%s]响应校验失败: %w", target, err)
	}
	return nil
}

func (n *alertCompletionNotifier) resolveBuiltinTemplateSetID(ctx context.Context,
	rule domain.ExecutionNotificationRule) (int64, error) {
	if n.templateClient == nil {
		return 0, fmt.Errorf("ETask 默认通知模板解析客户端不能为空")
	}
	for _, channel := range rule.Channels {
		if channel != domain.NotificationChannelLarkCard {
			return 0, fmt.Errorf("ETask 默认通知模板目前仅支持飞书卡片")
		}
	}
	response, err := n.templateClient.ResolveTemplateID(ctx, &templatev1.ResolveTemplateIDRequest{
		BizId:   int64(notificationv1.Business_TASK),
		Key:     builtinTaskExecutionTemplateSetKey,
		Channel: notificationv1.Channel_LARK_CARD,
	})
	if err != nil {
		return 0, fmt.Errorf("解析 ETask 默认通知模板集失败: %w", err)
	}
	if response == nil || response.GetTemplateSetId() <= 0 {
		return 0, fmt.Errorf("解析 ETask 默认通知模板集失败: 模板集不存在")
	}
	return response.GetTemplateSetId(), nil
}
