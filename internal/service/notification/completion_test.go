package notification

import (
	"context"
	"sync"
	"testing"

	notificationv1 "github.com/Duke1616/etask/api/proto/gen/ealert/notification/v1"
	templatev1 "github.com/Duke1616/etask/api/proto/gen/ealert/template/v1"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type notificationClientStub struct {
	notificationv1.NotificationServiceClient
	mu       sync.Mutex
	request  *notificationv1.DispatchNotificationRequest
	requests []*notificationv1.DispatchNotificationRequest
	calls    int
}

type templateClientStub struct {
	templatev1.TemplateServiceClient
	response *templatev1.ResolveTemplateIDResponse
}

func (s *templateClientStub) ResolveTemplateID(_ context.Context, _ *templatev1.ResolveTemplateIDRequest,
	_ ...grpc.CallOption) (*templatev1.ResolveTemplateIDResponse, error) {
	return s.response, nil
}

func (s *notificationClientStub) DispatchNotification(_ context.Context,
	req *notificationv1.DispatchNotificationRequest,
	_ ...grpc.CallOption) (*notificationv1.DispatchNotificationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.request = req
	s.requests = append(s.requests, req)
	return &notificationv1.DispatchNotificationResponse{
		Results: []*notificationv1.NotificationDispatchResult{{
			Channel: notificationv1.Channel_EMAIL, Status: notificationv1.SendStatus_SUCCEEDED,
		}},
	}, nil
}

func TestEAlertCompletionNotifierBuildsExecutionSnapshot(t *testing.T) {
	client := &notificationClientStub{}
	notifier := NewEAlertCompletionNotifier(client, nil)
	rule := domain.ExecutionNotificationRule{
		TriggerStatus: domain.TaskExecutionStatusSuccess, TemplateSetID: 42, Enabled: true,
		Recipients: []domain.NotificationRecipient{{
			Type: domain.NotificationRecipientUser, TargetIDs: []int64{101, 102},
		}},
		Channels: []domain.NotificationChannel{domain.NotificationChannelEmail},
	}
	execution := domain.TaskExecution{
		ID: 40, TenantID: 20, Source: domain.TaskExecutionSourceTask,
		Status: domain.TaskExecutionStatusSuccess, TaskResult: `{"changed":2}`,
		StartTime: 1000, EndTime: 2500, RetryCount: 1, ExecutorNodeID: "executor-1",
		Task: domain.Task{
			ID: 10, Name: "snapshot-name", Type: domain.TaskTypeRecurring,
			CronExpr: "0 * * * * *", ScheduleParams: map[string]string{"region": "cn"},
		},
	}

	err := notifier.Notify(t.Context(), rule, execution)

	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
	require.Equal(t, notificationv1.Business_TASK, client.request.GetBusiness())
	require.Equal(t, "etask:execution:40:completed", client.request.GetIdempotencyKey())
	require.Equal(t, int64(42), client.request.GetTemplateSetId())
	require.Empty(t, client.request.GetTemplateSetKey())
	require.Equal(t, []int64{101, 102}, client.request.GetRecipients()[0].GetTargetIds())
	params := client.request.GetTemplateParams().AsMap()
	require.Equal(t, "[任务通知] 任务 snapshot-name 执行成功", params["Subject"])
	require.Equal(t, "snapshot-name", params["Task"].(map[string]any)["Name"])
	require.Equal(t, "周期任务", params["Task"].(map[string]any)["TypeText"])
	require.Equal(t, "40", params["Execution"].(map[string]any)["ID"])
	require.Equal(t, "任务调度", params["Execution"].(map[string]any)["SourceText"])
	require.Equal(t, float64(1500), params["Execution"].(map[string]any)["DurationMillis"])
	require.Equal(t, "cn", params["ScheduleParams"].(map[string]any)["region"])
}

func TestEAlertCompletionNotifierRejectsMissingTaskSnapshot(t *testing.T) {
	client := &notificationClientStub{}
	notifier := NewEAlertCompletionNotifier(client, nil)
	rule := domain.ExecutionNotificationRule{TemplateSetID: 42, Enabled: true}

	err := notifier.Notify(t.Context(), rule, domain.TaskExecution{ID: 40})

	require.ErrorContains(t, err, "执行快照缺少任务 ID")
	require.Zero(t, client.calls)
}

func TestEAlertCompletionNotifierResolvesBuiltinTemplate(t *testing.T) {
	client := &notificationClientStub{}
	templateClient := &templateClientStub{response: &templatev1.ResolveTemplateIDResponse{TemplateSetId: 99}}
	notifier := NewEAlertCompletionNotifier(client, templateClient)
	rule := domain.ExecutionNotificationRule{
		TriggerStatus: domain.TaskExecutionStatusSuccess,
		TemplateSetID: 0,
		Recipients: []domain.NotificationRecipient{{
			Type: domain.NotificationRecipientUser, TargetIDs: []int64{101},
		}},
		Channels: []domain.NotificationChannel{domain.NotificationChannelLarkCard},
		Enabled:  true,
	}
	execution := domain.TaskExecution{
		ID:     40,
		Status: domain.TaskExecutionStatusSuccess,
		Task:   domain.Task{ID: 10, Name: "builtin-template"},
	}

	require.NoError(t, notifier.Notify(t.Context(), rule, execution))
	require.Equal(t, int64(99), client.request.GetTemplateSetId())
	require.Empty(t, client.request.GetTemplateSetKey())
}

func TestValidateDispatchNotificationResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *notificationv1.DispatchNotificationResponse
		wantErr  string
	}{
		{
			name:    "nil response",
			wantErr: "EAlert 返回空响应",
		},
		{
			name: "request error",
			response: &notificationv1.DispatchNotificationResponse{
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: "invalid template",
			},
			wantErr: "EAlert 处理任务通知失败",
		},
		{
			name: "empty channel result",
			response: &notificationv1.DispatchNotificationResponse{
				Results: []*notificationv1.NotificationDispatchResult{nil},
			},
			wantErr: "EAlert 返回空渠道结果",
		},
		{
			name: "channel error",
			response: &notificationv1.DispatchNotificationResponse{
				Results: []*notificationv1.NotificationDispatchResult{{
					Channel:      notificationv1.Channel_EMAIL,
					ErrorCode:    notificationv1.ErrorCode_SEND_NOTIFICATION_FAILED,
					ErrorMessage: "provider failed",
				}},
			},
			wantErr: "EAlert 渠道投递失败",
		},
		{
			name: "channel failed status",
			response: &notificationv1.DispatchNotificationResponse{
				Results: []*notificationv1.NotificationDispatchResult{{
					Channel: notificationv1.Channel_EMAIL,
					Status:  notificationv1.SendStatus_FAILED,
				}},
			},
			wantErr: "EAlert 渠道投递失败",
		},
		{
			name: "success",
			response: &notificationv1.DispatchNotificationResponse{
				Results: []*notificationv1.NotificationDispatchResult{{
					Channel: notificationv1.Channel_EMAIL,
					Status:  notificationv1.SendStatus_SUCCEEDED,
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDispatchNotificationResponse(test.response)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestEAlertCompletionNotifierDispatchBusinessNotifications(t *testing.T) {
	testCases := []struct {
		name         string
		ctx          func(t *testing.T) context.Context
		config       domain.BusinessNotificationConfig
		execution    domain.TaskExecution
		wantCalls    int
		wantErr      string
		validateReqs func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest)
	}{
		{
			name: "成功派发 notifications 列表中的全部目标用户",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelWechat},
				TemplateSetID:  88,
				IdentifierType: domain.RecipientIdentifierTypeUserID,
			},
			execution: domain.TaskExecution{
				ID: 101,
				Task: domain.Task{
					ID:   12,
					Name: "ad-password-scanner",
				},
				TaskResult: `{"total":2,"notifications":[{"user_id":1001,"username":"zhangsan","days_remaining":3},{"user_id":1002,"username":"lisi","days_remaining":5}]}`,
			},
			wantCalls: 2,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 2)
				reqMap := make(map[string]*notificationv1.DispatchNotificationRequest)
				for _, r := range reqs {
					p := r.GetTemplateParams().AsMap()
					if uname, ok := p["username"].(string); ok {
						reqMap[uname] = r
					}
				}

				// 校验 zhangsan
				req1, ok := reqMap["zhangsan"]
				require.True(t, ok)
				require.Equal(t, int64(88), req1.GetTemplateSetId())
				require.Equal(t, []int64{1001}, req1.GetRecipients()[0].GetTargetIds())
				params1 := req1.GetTemplateParams().AsMap()
				require.Equal(t, float64(3), params1["days_remaining"])
				require.Equal(t, "ad-password-scanner 业务提醒", params1["Title"])

				// 校验 lisi
				req2, ok := reqMap["lisi"]
				require.True(t, ok)
				require.Equal(t, []int64{1002}, req2.GetRecipients()[0].GetTargetIds())
				params2 := req2.GetTemplateParams().AsMap()
				require.Equal(t, float64(5), params2["days_remaining"])
			},
		},
		{
			name: "STATIC 固定指定模式下将业务产物整体投递给预设人员",
			config: domain.BusinessNotificationConfig{
				Enabled:       true,
				Mode:          domain.BusinessRecipientModeStatic,
				Channels:      []domain.NotificationChannel{domain.NotificationChannelLarkCard},
				TemplateSetID: 66,
				Recipients: []domain.NotificationRecipient{{
					Type:      domain.NotificationRecipientTeam,
					TargetIDs: []int64{999},
				}},
			},
			execution: domain.TaskExecution{
				ID: 110,
				Task: domain.Task{
					ID:   30,
					Name: "daily-audit-report",
				},
				TaskResult: `{"notifications":[{"target":"user1","status":"warning"},{"target":"user2","status":"expired"}]}`,
			},
			wantCalls: 1,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 1)
				req := reqs[0]
				require.Equal(t, int64(66), req.GetTemplateSetId())
				require.Equal(t, "etask:execution:110:biz_notify:static", req.GetIdempotencyKey())
				require.Len(t, req.GetRecipients(), 1)
				require.Equal(t, []int64{999}, req.GetRecipients()[0].GetTargetIds())
				params := req.GetTemplateParams().AsMap()
				require.Equal(t, float64(2), params["ItemCount"])
				require.Equal(t, "daily-audit-report 业务数据产出提醒", params["Title"])
			},
		},
		{
			name: "STATIC 固定指定模式下无明细条目但有汇总指标时正常派发并扁平合并指标",
			config: domain.BusinessNotificationConfig{
				Enabled:       true,
				Mode:          domain.BusinessRecipientModeStatic,
				Channels:      []domain.NotificationChannel{domain.NotificationChannelLarkCard},
				TemplateSetID: 66,
				Recipients: []domain.NotificationRecipient{{
					Type:      domain.NotificationRecipientUser,
					TargetIDs: []int64{2001},
				}},
			},
			execution: domain.TaskExecution{
				ID: 111,
				Task: domain.Task{
					ID:   31,
					Name: "summary-only-task",
				},
				TaskResult: `{"total_scanned":150,"expired_count":0,"status":"healthy"}`,
			},
			wantCalls: 1,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 1)
				req := reqs[0]
				params := req.GetTemplateParams().AsMap()
				require.Equal(t, float64(0), params["ItemCount"])
				require.Equal(t, float64(150), params["total_scanned"])
				require.Equal(t, float64(0), params["expired_count"])
				require.Equal(t, "healthy", params["status"])
			},
		},
		{
			name: "未启用配置时不触发派发",
			config: domain.BusinessNotificationConfig{
				Enabled: false,
			},
			execution: domain.TaskExecution{
				ID:         102,
				TaskResult: `{"notifications":[{"user_id":1001}]}`,
			},
			wantCalls: 0,
		},
		{
			name: "支持嵌套 params 结构体格式派发",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelLarkCard},
				TemplateSetID:  99,
				IdentifierType: domain.RecipientIdentifierTypeUsername,
			},
			execution: domain.TaskExecution{
				ID: 104,
				Task: domain.Task{
					ID:   15,
					Name: "db-cert-checker",
				},
				TaskResult: `{"notifications":[{"recipient":"admin_user","params":{"cert_name":"ssl-gateway","days":7}}]}`,
			},
			wantCalls: 1,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 1)
				req := reqs[0]
				require.Equal(t, int64(99), req.GetTemplateSetId())
				require.Equal(t, "etask:execution:104:biz_notify:0:admin_user", req.GetIdempotencyKey())
				require.Equal(t, notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME, req.GetRecipients()[0].GetIdentifierType())
				require.Equal(t, []string{"admin_user"}, req.GetRecipients()[0].GetValues())
				params := req.GetTemplateParams().AsMap()
				require.Equal(t, "admin_user", params["Target"])
				require.Equal(t, "ssl-gateway", params["cert_name"])
				require.Equal(t, float64(7), params["days"])
			},
		},
		{
			name: "支持使用 target 与 user_id 字段并发分发",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelWechat},
				TemplateSetID:  77,
				IdentifierType: domain.RecipientIdentifierTypeUserID,
			},
			execution: domain.TaskExecution{
				ID: 105,
				Task: domain.Task{
					ID:   18,
					Name: "ad-password-scanner",
				},
				TaskResult: `{"notifications":[{"target":"wangwu","user_id":2001,"days":2},{"target":"zhaoliu","user_id":2002,"days":4}]}`,
			},
			wantCalls: 2,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 2)
			},
		},
		{
			name: "结果中 notifications 列表为空时不触发派发",
			config: domain.BusinessNotificationConfig{
				Enabled:       true,
				Channels:      []domain.NotificationChannel{domain.NotificationChannelWechat},
				TemplateSetID: 88,
			},
			execution: domain.TaskExecution{
				ID: 103,
				Task: domain.Task{
					ID:   12,
					Name: "ad-password-scanner",
				},
				TaskResult: `{"total":0,"notifications":[]}`,
			},
			wantCalls: 0,
		},
		{
			name: "Context 已取消时立即中断分发并返回 context canceled 错误",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelWechat},
				TemplateSetID:  88,
				IdentifierType: domain.RecipientIdentifierTypeUserID,
			},
			execution: domain.TaskExecution{
				ID: 104,
				Task: domain.Task{
					ID:   12,
					Name: "ad-password-scanner",
				},
				TaskResult: `{"notifications":[{"target":"2001"},{"target":"2002"}]}`,
			},
			wantCalls: 0,
			wantErr:   "context canceled",
		},
		{
			name: "显式声明 USERNAME 直接构建 RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME 委托 EAlert 解析",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelLarkCard},
				TemplateSetID:  99,
				IdentifierType: domain.RecipientIdentifierTypeUsername,
			},
			execution: domain.TaskExecution{
				ID: 106,
				Task: domain.Task{
					ID:   20,
					Name: "ad-password-scanner",
				},
				TaskResult: `{"notifications":[{"target":"langtt","days":10},{"target":"wangq","days":-27}]}`,
			},
			wantCalls: 2,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 2)
				targetMap := make(map[string]*notificationv1.RecipientSelector)
				for _, r := range reqs {
					target := r.GetTemplateParams().AsMap()["Target"].(string)
					require.Len(t, r.GetRecipients(), 1)
					targetMap[target] = r.GetRecipients()[0]
				}

				require.Equal(t, notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME, targetMap["langtt"].GetIdentifierType())
				require.Equal(t, []string{"langtt"}, targetMap["langtt"].GetValues())

				require.Equal(t, notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME, targetMap["wangq"].GetIdentifierType())
				require.Equal(t, []string{"wangq"}, targetMap["wangq"].GetValues())
			},
		},
		{
			name: "声明为 USER_ID 走老字段 TargetIds 100% 向后兼容",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelLarkCard},
				TemplateSetID:  99,
				IdentifierType: domain.RecipientIdentifierTypeUserID,
			},
			execution: domain.TaskExecution{
				ID: 108,
				Task: domain.Task{
					ID:   22,
					Name: "user-id-task",
				},
				TaskResult: `{"notifications":[{"target":"1001"}]}`,
			},
			wantCalls: 1,
			validateReqs: func(t *testing.T, reqs []*notificationv1.DispatchNotificationRequest) {
				require.Len(t, reqs, 1)
				require.Equal(t, []int64{1001}, reqs[0].GetRecipients()[0].GetTargetIds())
			},
		},
		{
			name: "声明为 USER_ID 但接收人非正整数时明确报错",
			config: domain.BusinessNotificationConfig{
				Enabled:        true,
				Channels:       []domain.NotificationChannel{domain.NotificationChannelWechat},
				TemplateSetID:  88,
				IdentifierType: domain.RecipientIdentifierTypeUserID,
			},
			execution: domain.TaskExecution{
				ID: 107,
				Task: domain.Task{
					ID:   21,
					Name: "bad-task",
				},
				TaskResult: `{"notifications":[{"target":"not-a-number"}]}`,
			},
			wantCalls: 0,
			wantErr:   "非合法的正整数用户ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &notificationClientStub{}
			notifier := NewEAlertCompletionNotifier(client, nil)

			ctx := t.Context()
			if tc.ctx != nil {
				ctx = tc.ctx(t)
			}

			err := notifier.DispatchBusinessNotifications(ctx, tc.config, tc.execution)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			require.Equal(t, tc.wantCalls, client.calls)

			if tc.validateReqs != nil {
				tc.validateReqs(t, client.requests)
			}
		})
	}
}

func TestBuildBusinessRecipientSelector(t *testing.T) {
	testCases := []struct {
		name           string
		identifierType domain.RecipientIdentifierType
		item           BusinessNotificationItem
		wantSelector   *notificationv1.RecipientSelector
		wantErr        string
	}{
		{
			name:           "USERNAME 模式正常解析",
			identifierType: domain.RecipientIdentifierTypeUsername,
			item:           BusinessNotificationItem{Target: "langtt"},
			wantSelector: &notificationv1.RecipientSelector{
				Type:           notificationv1.RecipientSelectorType_RECIPIENT_USER,
				IdentifierType: notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME,
				Values:         []string{"langtt"},
			},
		},
		{
			name:           "USERNAME 模式目标为空时报错",
			identifierType: domain.RecipientIdentifierTypeUsername,
			item:           BusinessNotificationItem{Target: ""},
			wantErr:        "目标用户名为空",
		},
		{
			name:           "USER_ID 模式正常解析正整数",
			identifierType: domain.RecipientIdentifierTypeUserID,
			item:           BusinessNotificationItem{Target: "8888"},
			wantSelector: &notificationv1.RecipientSelector{
				Type:      notificationv1.RecipientSelectorType_RECIPIENT_USER,
				TargetIds: []int64{8888},
			},
		},
		{
			name:           "USER_ID 模式非数字报错",
			identifierType: domain.RecipientIdentifierTypeUserID,
			item:           BusinessNotificationItem{Target: "not-numeric"},
			wantErr:        "非合法的正整数用户ID",
		},
		{
			name:           "默认模式数字自动走 TargetIds",
			identifierType: "",
			item:           BusinessNotificationItem{Target: "666"},
			wantSelector: &notificationv1.RecipientSelector{
				Type:      notificationv1.RecipientSelectorType_RECIPIENT_USER,
				TargetIds: []int64{666},
			},
		},
		{
			name:           "默认模式非数字自动走 USERNAME",
			identifierType: "",
			item:           BusinessNotificationItem{Target: "wangwu"},
			wantSelector: &notificationv1.RecipientSelector{
				Type:           notificationv1.RecipientSelectorType_RECIPIENT_USER,
				IdentifierType: notificationv1.RecipientIdentifierType_RECIPIENT_IDENTIFIER_USERNAME,
				Values:         []string{"wangwu"},
			},
		},
		{
			name:           "默认模式目标为空报错",
			identifierType: "",
			item:           BusinessNotificationItem{Target: ""},
			wantErr:        "缺少有效的接收人标识",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector, err := buildBusinessRecipientSelector(tc.identifierType, tc.item)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantSelector, selector)
		})
	}
}
