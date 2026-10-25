package notification

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	notificationv1 "github.com/Duke1616/etask/api/proto/gen/ealert/notification/v1"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/structpb"
)

// buildDispatchNotificationRequest 将领域通知规则和执行快照组装为 EAlert 请求。
func buildDispatchNotificationRequest(rule domain.ExecutionNotificationRule,
	execution domain.TaskExecution, templateSetID int64) (*notificationv1.DispatchNotificationRequest, error) {
	if err := validateExecutionSnapshot(execution); err != nil {
		return nil, err
	}
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("任务执行通知规则无效: %w", err)
	}
	if templateSetID <= 0 {
		return nil, fmt.Errorf("任务执行通知缺少有效的模板集 ID")
	}

	templateParams, err := buildCompletionTemplateParams(execution)
	if err != nil {
		return nil, err
	}

	return &notificationv1.DispatchNotificationRequest{
		Business:       notificationv1.Business_TASK,
		IdempotencyKey: fmt.Sprintf("etask:execution:%d:completed", execution.ID),
		Recipients:     toProtoRecipients(rule.Recipients),
		Channels:       toProtoChannels(rule.Channels),
		TemplateSetId:  templateSetID,
		TemplateParams: templateParams,
	}, nil
}

// validateExecutionSnapshot 检查构造通知所需的最小执行快照。
func validateExecutionSnapshot(execution domain.TaskExecution) error {
	if execution.ID <= 0 {
		return fmt.Errorf("执行快照缺少有效的执行 ID: %d", execution.ID)
	}
	if execution.Task.ID <= 0 {
		return fmt.Errorf("执行快照缺少任务 ID: execution_id=%d", execution.ID)
	}
	return nil
}

// buildBaseTemplateParams 构造所有通知共有的基础上下文参数（标题、执行与任务快照）。
func buildBaseTemplateParams(execution domain.TaskExecution, title, subjectPrefix string) map[string]any {
	if subjectPrefix == "" {
		subjectPrefix = "[任务通知]"
	}
	return map[string]any{
		"Title":        title,
		"Subject":      fmt.Sprintf("%s %s", subjectPrefix, title),
		"MessageTitle": fmt.Sprintf("📢 %s %s", subjectPrefix, title),
		"Task":         buildTaskTemplateParams(execution.Task),
		"Execution":    buildExecutionTemplateParams(execution),
	}
}

// buildCompletionTemplateParams 构造传递给 EAlert 模板的稳定参数。
func buildCompletionTemplateParams(execution domain.TaskExecution) (*structpb.Struct, error) {
	title := executionNotificationTitle(execution.Task.Name, execution.Status)
	paramsMap := buildBaseTemplateParams(execution, title, "[任务通知]")
	paramsMap["ScheduleParams"] = stringMapToAny(execution.Task.ScheduleParams)

	params, err := structpb.NewStruct(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("构造 EAlert 模板参数失败: %w", err)
	}
	return params, nil
}

// TaskResultPayload 描述任务执行写回 FD3 的平台消费契约。
type TaskResultPayload struct {
	Notifications []BusinessNotificationItem `json:"notifications"`
}

// BusinessNotificationItem 描述单条待投递的业务通知结构体。
type BusinessNotificationItem struct {
	Target string         `json:"target"` // 接收人标识（如用户名或数值用户 ID）
	Params map[string]any `json:"params"` // 传给 EAlert 模板渲染的业务参数
}

// UnmarshalJSON 支持直接提取 target（或历史字段 recipient/user_id）以及全部模板参数。
func (item *BusinessNotificationItem) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for _, key := range []string{"target", "recipient", "user_id", "username"} {
		if val, ok := raw[key]; ok && val != nil && val != "" {
			item.Target = fmt.Sprintf("%v", val)
			break
		}
	}

	if paramsRaw, ok := raw["params"].(map[string]any); ok && paramsRaw != nil {
		item.Params = paramsRaw
	} else {
		item.Params = raw
	}
	return nil
}

// parseTaskResultPayload 反序列化执行结果中的通知载荷。
func parseTaskResultPayload(taskResult string) (*TaskResultPayload, error) {
	if taskResult == "" {
		return nil, nil
	}
	var payload TaskResultPayload
	if err := json.Unmarshal([]byte(taskResult), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// buildBusinessNotificationRequest 构造单条业务通知的 EAlert 投递请求。
func buildBusinessNotificationRequest(config domain.BusinessNotificationConfig, execution domain.TaskExecution,
	itemIndex int, item BusinessNotificationItem) (*notificationv1.DispatchNotificationRequest, error) {
	itemParams, err := buildDynamicItemTemplateParams(execution, item.Params, item.Target)
	if err != nil {
		return nil, err
	}

	selector, err := buildBusinessRecipientSelector(config.IdentifierType, item)
	if err != nil {
		return nil, fmt.Errorf("条目[%d]%w", itemIndex, err)
	}

	idempotencyKey := fmt.Sprintf("etask:execution:%d:biz_notify:%d:%s", execution.ID, itemIndex, item.Target)
	return &notificationv1.DispatchNotificationRequest{
		Business:       notificationv1.Business_TASK,
		IdempotencyKey: idempotencyKey,
		Recipients:     []*notificationv1.RecipientSelector{selector},
		Channels:       toProtoChannels(config.Channels),
		TemplateSetId:  config.TemplateSetID,
		TemplateParams: itemParams,
	}, nil
}

// buildStaticBusinessNotificationRequest 构造静态指定接收人模式下的 EAlert 汇总通知请求。
func buildStaticBusinessNotificationRequest(config domain.BusinessNotificationConfig, execution domain.TaskExecution,
	payload *TaskResultPayload) (*notificationv1.DispatchNotificationRequest, error) {
	title := fmt.Sprintf("%s 业务数据产出提醒", execution.Task.Name)
	paramsMap := buildBaseTemplateParams(execution, title, "[业务通知]")

	itemCount := 0
	var items []any
	if payload != nil {
		itemCount = len(payload.Notifications)
		items = lo.Map(payload.Notifications, func(n BusinessNotificationItem, _ int) any {
			return map[string]any{
				"target": n.Target,
				"params": n.Params,
			}
		})
	}
	paramsMap["ItemCount"] = itemCount
	paramsMap["Notifications"] = items

	// 将脚本返回的顶层 JSON 字段扁平合并到模板根参数中，方便模板直接使用指标变量
	if resultData := parseResultData(execution.TaskResult); resultData != nil {
		if resultMap, ok := resultData.(map[string]any); ok {
			for k, v := range resultMap {
				if _, exists := paramsMap[k]; !exists {
					paramsMap[k] = v
				}
			}
		}
	}

	params, err := structpb.NewStruct(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("构造业务通知模板参数失败: %w", err)
	}

	idempotencyKey := fmt.Sprintf("etask:execution:%d:biz_notify:static", execution.ID)
	return &notificationv1.DispatchNotificationRequest{
		Business:       notificationv1.Business_TASK,
		IdempotencyKey: idempotencyKey,
		Recipients:     toProtoRecipients(config.Recipients),
		Channels:       toProtoChannels(config.Channels),
		TemplateSetId:  config.TemplateSetID,
		TemplateParams: params,
	}, nil
}

// buildBusinessRecipientSelector 根据配置的标识维度与条目信息构建 EAlert 接收人选择器。
func buildBusinessRecipientSelector(identifierType domain.RecipientIdentifierType,
	item BusinessNotificationItem) (*notificationv1.RecipientSelector, error) {
	selector := &notificationv1.RecipientSelector{
		Type: notificationv1.RecipientSelectorType_RECIPIENT_USER,
	}

	switch identifierType {
	case domain.RecipientIdentifierTypeUsername:
		if item.Target == "" {
			return nil, fmt.Errorf("目标用户名为空")
		}
		selector.IdentifierType = notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME
		selector.Values = []string{item.Target}
		return selector, nil

	case domain.RecipientIdentifierTypeUserID:
		id, err := parseSingleNumericTargetID(item)
		if err != nil {
			return nil, fmt.Errorf("目标[%s]非合法的正整数用户ID", item.Target)
		}
		selector.TargetIds = []int64{id}
		return selector, nil

	default:
		// 历史配置或未显式配置时向后兼容：纯数字使用 TargetIds，非纯数字使用 USERNAME
		if id, err := parseSingleNumericTargetID(item); err == nil && id > 0 {
			selector.TargetIds = []int64{id}
			return selector, nil
		}
		if item.Target != "" {
			selector.IdentifierType = notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME
			selector.Values = []string{item.Target}
			return selector, nil
		}
		return nil, fmt.Errorf("缺少有效的接收人标识")
	}
}

func parseSingleNumericTargetID(item BusinessNotificationItem) (int64, error) {
	id, err := strconv.ParseInt(item.Target, 10, 64)
	if err != nil && item.Params != nil {
		if uid, ok := item.Params["user_id"]; ok && uid != nil {
			id, err = strconv.ParseInt(fmt.Sprintf("%v", uid), 10, 64)
		}
	}
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("无效的正整数用户ID: %s", item.Target)
	}
	return id, nil
}

// buildDynamicItemTemplateParams 构造单条动态派发条目的专属模板参数。
func buildDynamicItemTemplateParams(execution domain.TaskExecution, itemMap map[string]any,
	targetVal any) (*structpb.Struct, error) {
	title := fmt.Sprintf("%s 业务提醒", execution.Task.Name)
	payload := buildBaseTemplateParams(execution, title, "[业务通知]")
	payload["Target"] = fmt.Sprintf("%v", targetVal)
	payload["Item"] = itemMap

	// 扁平合并条目原生字段，便于模板直接按变量名（如 .days_remaining）插值取用
	for k, v := range itemMap {
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
	}

	return structpb.NewStruct(payload)
}

// buildTaskTemplateParams 构造不包含敏感配置的任务模板快照。
func buildTaskTemplateParams(task domain.Task) map[string]any {
	return map[string]any{
		"ID":       strconv.FormatInt(task.ID, 10),
		"Name":     task.Name,
		"Type":     task.Type.String(),
		"TypeText": taskTypeText(task.Type),
		"CronExpr": task.CronExpr,
	}
}

// buildExecutionTemplateParams 构造执行状态、时间和结果模板快照。
func buildExecutionTemplateParams(execution domain.TaskExecution) map[string]any {
	durationMillis := executionDurationMillis(execution)
	return map[string]any{
		"ID":             strconv.FormatInt(execution.ID, 10),
		"Status":         execution.Status.String(),
		"StatusText":     executionStatusText(execution.Status),
		"Result":         execution.TaskResult,
		"ResultData":     parseResultData(execution.TaskResult),
		"Source":         execution.Source.String(),
		"SourceText":     executionSourceText(execution.Source),
		"RequestID":      execution.RequestID,
		"StartTime":      execution.StartTime,
		"StartedAt":      formatUnixMillis(execution.StartTime),
		"EndTime":        execution.EndTime,
		"EndedAt":        formatUnixMillis(execution.EndTime),
		"DurationMillis": durationMillis,
		"Duration":       (time.Duration(durationMillis) * time.Millisecond).String(),
		"RetryCount":     execution.RetryCount,
		"ExecutorNodeID": execution.ExecutorNodeID,
	}
}

// executionDurationMillis 返回执行耗时，并将异常的负数时间差归零。
func executionDurationMillis(execution domain.TaskExecution) int64 {
	if execution.EndTime <= execution.StartTime {
		return 0
	}
	return execution.EndTime - execution.StartTime
}

// executionNotificationTitle 生成邮件、飞书等渠道共用的通知标题。
func executionNotificationTitle(taskName string, status domain.TaskExecutionStatus) string {
	return fmt.Sprintf("任务 %s 执行%s", taskName, executionStatusText(status))
}

// executionStatusText 返回面向用户的执行状态文案。
func executionStatusText(status domain.TaskExecutionStatus) string {
	switch status {
	case domain.TaskExecutionStatusSuccess:
		return "成功"
	case domain.TaskExecutionStatusFailed:
		return "失败"
	case domain.TaskExecutionStatusCancelled:
		return "已取消"
	default:
		return status.String()
	}
}

// taskTypeText 返回面向用户的任务类型文案。
func taskTypeText(taskType domain.TaskType) string {
	switch taskType {
	case domain.TaskTypeRecurring:
		return "周期任务"
	case domain.TaskTypeOneTime:
		return "一次性任务"
	default:
		return taskType.String()
	}
}

// executionSourceText 返回面向用户的执行来源文案。
func executionSourceText(source domain.TaskExecutionSource) string {
	switch source {
	case domain.TaskExecutionSourceTask:
		return "任务调度"
	case domain.TaskExecutionSourceCodebookPreview:
		return "Codebook 试运行"
	case domain.TaskExecutionSourceWorkflow:
		return "工作流"
	default:
		return source.String()
	}
}

// formatUnixMillis 将毫秒时间戳格式化为本地通知时间。
func formatUnixMillis(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.UnixMilli(value).In(time.Local).Format("2006-01-02 15:04:05")
}

// parseResultData 仅解析合法 JSON，供飞书等结构化模板使用。
func parseResultData(result string) any {
	if result == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		return nil
	}
	return value
}

// stringMapToAny 将调度参数转换为 Struct 可接受的值类型。
func stringMapToAny(src map[string]string) map[string]any {
	return lo.MapValues(src, func(value string, _ string) any {
		return value
	})
}
