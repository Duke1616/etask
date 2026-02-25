-- +goose Up
-- +goose StatementBegin
ALTER TABLE task_executions
  MODIFY COLUMN status ENUM(
    'WAITING_PULL',
    'PREPARE',
    'RUNNING',
    'FAILED_RETRYABLE',
    'FAILED_RESCHEDULED',
    'FAILED',
    'SUCCESS'
  ) NOT NULL DEFAULT 'PREPARE'
  COMMENT '执行状态: WAITING_PULL-等待节点拉取, PREPARE-初始化, RUNNING-执行中, FAILED_RETRYABLE-可重试失败, FAILED_RESCHEDULED-重调度失败, FAILED-失败, SUCCESS-成功';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE task_executions
  MODIFY COLUMN status ENUM(
    'PREPARE',
    'RUNNING',
    'FAILED_RETRYABLE',
    'FAILED_RESCHEDULED',
    'FAILED',
    'SUCCESS'
  ) NOT NULL DEFAULT 'PREPARE';
-- +goose StatementEnd

