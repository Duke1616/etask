package ioc

import (
	executorv1 "github.com/Duke1616/etask/api/proto/gen/etask/executor/v1"
	"github.com/Duke1616/etask/internal/compensator"
	"github.com/Duke1616/etask/internal/service/runner"
	"github.com/Duke1616/etask/internal/service/task"
	"github.com/Duke1616/etask/pkg/grpc/pool"
	"github.com/spf13/viper"
)

func InitRetryCompensator(
	runner runner.Runner,
	execSvc task.ExecutionService,
) *compensator.RetryCompensator {
	var cfg compensator.RetryConfig
	err := viper.UnmarshalKey("compensator.retry", &cfg)
	if err != nil {
		panic(err)
	}
	return compensator.NewRetryCompensator(
		runner,
		execSvc,
		cfg,
	)
}

func InitRescheduleCompensator(
	runner runner.Runner,
	execSvc task.ExecutionService,
) *compensator.RescheduleCompensator {
	var cfg compensator.RescheduleConfig
	err := viper.UnmarshalKey("compensator.reschedule", &cfg)
	if err != nil {
		panic(err)
	}
	return compensator.NewRescheduleCompensator(
		runner,
		execSvc,
		cfg)
}

func InitInterruptCompensator(
	grpcClients *pool.Clients[executorv1.ExecutorServiceClient],
	execSvc task.ExecutionService,
) *compensator.InterruptCompensator {
	var cfg compensator.InterruptConfig
	err := viper.UnmarshalKey("compensator.interrupt", &cfg)
	if err != nil {
		panic(err)
	}
	return compensator.NewInterruptCompensator(
		grpcClients,
		execSvc,
		cfg,
	)
}
