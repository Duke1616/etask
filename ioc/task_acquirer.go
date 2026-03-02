package ioc

import (
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/service/acquirer"
)

func InitMySQLTaskAcquirer(taskRepo repository.TaskRepository) acquirer.TaskAcquirer {
	return acquirer.NewTaskAcquirer(taskRepo)
}
