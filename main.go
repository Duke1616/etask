package main

import (
	"context"
	"fmt"
	"os"

	scheduler_ioc "github.com/Duke1616/ework-runner/cmd/scheduler/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// 1. 初始化 Viper 配置
	initViper()

	// 2. 创建 Ego 应用实例
	egoApp := ego.New()

	// 3. 初始化全量应用（对等拓扑，虽然有多种角色，但二进制文件是统一的）
	app := scheduler_ioc.InitSchedulerApp()

	// 4. 根据 app.mode 控制组件开启与关闭 (all | scheduler | agent)
	mode := viper.GetString("app.mode")
	if mode == "" {
		mode = "scheduler" // 默认调度模式
	}

	elog.Info("app_start",
		elog.String("mode", mode),
		elog.String("info", "starting ework-runner node"))

	ctx := context.Background()

	// 5. 分模式逻辑分发
	switch mode {
	case "scheduler":
		// 仅作为调度中心启动后台任务和补偿逻辑
		app.StartSchedulerTasks(ctx)
		// 注册对外暴露的 Web/gRPC 服务
		egoApp.Serve(
			app.Web,
			app.Server,
			app.Scheduler,
		)
	case "agent":
		// 仅作为执行代理启动 Kafka 任务监听
		app.StartAgent(ctx)
		egoApp.Serve(
			app.Web,
		)
	default: // all: 全能对等节点
		app.StartTasks(ctx)
		egoApp.Serve(
			app.Web,
			app.Server,
			app.Scheduler,
		)
	}

	// 6. 运行 Ego 管理的所有服务的生命周期
	if err := egoApp.Run(); err != nil {
		elog.Panic("app_run_error", elog.FieldErr(err))
	}
}

func initViper() {
	dir, _ := os.Getwd()
	// 支持通过命令行参数 --config 指定路径，默认为 config/prod.yaml
	file := pflag.String("config", dir+"/config/scheduler.yaml", "配置文件路径")
	pflag.Parse()

	viper.SetConfigFile(*file)
	viper.WatchConfig()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal_error_config_file: %w", err))
	}
}
