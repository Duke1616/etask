-- +goose Up
-- +goose StatementBegin
ALTER TABLE `task_executions` ADD INDEX `idx_task_id_ctime` (`task_id`, `ctime`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `task_executions` DROP INDEX `idx_task_id_ctime`;
-- +goose StatementEnd
