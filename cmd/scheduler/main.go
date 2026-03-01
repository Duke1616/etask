package main

import (
	"context"
	"os"

	"github.com/Duke1616/ework-runner/cmd/endpoint"
	"github.com/Duke1616/ework-runner/cmd/scheduler/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()

	rootCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "ework-runner scheduler",
		// 如果不输入子命令，就默认执行服务器
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "启动调度器服务",
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}

	// 注册子命令
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(endpoint.Cmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startServer() {
	// 创建 ego 应用实例
	egoApp := ego.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := ioc.InitSchedulerApp()
	app.StartTasks(ctx)

	// 启动服务 (显式注册调度中心需要的组件)
	if err := egoApp.Serve(app.Web, app.Server, app.Scheduler).
		Cron().
		Run(); err != nil {
		elog.Panic("startup", elog.FieldErr(err))
	}
}

func initViper() {
	dir, _ := os.Getwd()

	file := pflag.String(
		"config",
		dir+"/../../config/config.yaml",
		"配置文件路径",
	)
	pflag.Parse()

	viper.SetConfigFile(*file)

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}
