package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/domain"
)

type prepareResult struct {
	codebook  domain.Codebook
	runner    domain.Runner
	args      string
	timeout   int64
	variables []previewVariable
}

func (s *service) prepare(ctx context.Context, command RunCommand) (prepareResult, error) {
	if err := validateCommand(command); err != nil {
		return prepareResult{}, err
	}
	codebook, err := s.resolveCodebook(ctx, command.CodebookID)
	if err != nil {
		return prepareResult{}, err
	}
	runner, err := s.resolveRunner(ctx, command.RunnerID, codebook.ID)
	if err != nil {
		return prepareResult{}, err
	}
	args, err := normalizeArgs(command.Args)
	if err != nil {
		return prepareResult{}, err
	}
	timeout, err := normalizeTimeout(command.MaxExecutionSeconds)
	if err != nil {
		return prepareResult{}, err
	}
	variables, err := s.mergeVariables(ctx, runner.ID, command.Variables)
	if err != nil {
		return prepareResult{}, err
	}
	return prepareResult{codebook: codebook, runner: runner, args: args, timeout: timeout, variables: variables}, nil
}

func validateCommand(command RunCommand) error {
	if command.CodebookID <= 0 || command.RunnerID <= 0 {
		return fmt.Errorf("Codebook ID 和执行单元 ID 必须大于 0")
	}
	if strings.TrimSpace(command.Code) == "" {
		return fmt.Errorf("试运行代码不能为空")
	}
	return nil
}

func (s *service) resolveCodebook(ctx context.Context, id int64) (domain.Codebook, error) {
	codebook, err := s.codebookSvc.GetByID(ctx, id)
	if err != nil {
		return domain.Codebook{}, fmt.Errorf("查询 Codebook 文件失败: %w", err)
	}
	if !codebook.IsFile() {
		return domain.Codebook{}, fmt.Errorf("只有 Codebook 文件可以试运行")
	}
	return codebook, nil
}

func (s *service) resolveRunner(ctx context.Context, id, codebookID int64) (domain.Runner, error) {
	runner, err := s.runnerSvc.FindByID(ctx, id)
	if err != nil {
		return domain.Runner{}, fmt.Errorf("查询执行单元失败: %w", err)
	}
	if runner.CodebookID != codebookID {
		return domain.Runner{}, fmt.Errorf("执行单元未绑定当前 Codebook 文件")
	}
	if !runner.Kind.IsValid() || (runner.Handler != "python" && runner.Handler != "shell") {
		return domain.Runner{}, fmt.Errorf("试运行仅支持 Python 和 Shell 执行单元")
	}
	if runner.Action != domain.RunnerActionRegistered {
		return domain.Runner{}, fmt.Errorf("当前执行单元未启用")
	}
	return runner, nil
}

func normalizeArgs(raw string) (string, error) {
	args := strings.TrimSpace(raw)
	if args == "" {
		return "{}", nil
	}
	if !json.Valid([]byte(args)) {
		return "", fmt.Errorf("试运行参数必须是合法 JSON")
	}
	return args, nil
}

func normalizeTimeout(seconds int64) (int64, error) {
	if seconds == 0 {
		seconds = defaultTimeoutSeconds
	}
	if seconds < 1 || seconds > maxTimeoutSeconds {
		return 0, fmt.Errorf("试运行超时必须在 1 到 %d 秒之间", maxTimeoutSeconds)
	}
	return seconds, nil
}

// mergeVariables 保留默认变量顺序；临时变量覆盖同名值，新变量追加到末尾。
func (s *service) mergeVariables(ctx context.Context, runnerID int64,
	overrides []domain.RunnerVariable) ([]previewVariable, error) {
	defaults, err := s.runnerSvc.ListMergedVariables(ctx, runnerID)
	if err != nil {
		return nil, fmt.Errorf("查询执行单元变量失败: %w", err)
	}
	values := make(map[string]domain.RunnerVariable, len(defaults)+len(overrides))
	keys := make([]string, 0, len(defaults)+len(overrides))
	for _, variable := range defaults {
		values[variable.Key] = variable
		keys = append(keys, variable.Key)
	}
	for _, variable := range overrides {
		variable.Key = strings.TrimSpace(variable.Key)
		if variable.Key == "" {
			return nil, fmt.Errorf("临时变量名称不能为空")
		}
		if _, exists := values[variable.Key]; !exists {
			keys = append(keys, variable.Key)
		}
		values[variable.Key] = variable
	}
	result := make([]previewVariable, 0, len(keys))
	for _, key := range keys {
		variable := values[key]
		result = append(result, previewVariable{Key: key, Value: variable.Value, Secret: variable.Secret})
	}
	return result, nil
}

func (s *service) buildDraft(command RunCommand, prepared prepareResult,
	variablesJSON []byte) domain.TaskExecution {
	return domain.TaskExecution{
		Status: domain.TaskExecutionStatusPrepare, StartTime: time.Now().UnixMilli(),
		Task: domain.Task{
			Name:                "试运行: " + prepared.codebook.Name,
			MaxExecutionSeconds: prepared.timeout,
			RetryConfig:         &domain.RetryConfig{MaxRetries: 0},
			GrpcConfig: &domain.GrpcConfig{
				ServiceName: prepared.runner.Target, HandlerName: prepared.runner.Handler,
				Params: map[string]string{
					"code": command.Code, "args": prepared.args, "variables": string(variablesJSON),
				},
			},
		},
	}
}

type previewVariable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}
