package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Duke1616/etask/cmd/endpoint"
	"github.com/Duke1616/etask/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	modes   []string
	cfgFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ework-runner",
		Short: "ework-runner 统一入口",
	}

	dir, _ := os.Getwd()
	defaultCfg := dir + "/config/scheduler.yaml"
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultCfg, "配置文件路径")

	cobra.OnInitialize(initViper)

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "启动服务节点",
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}

	serverCmd.Flags().StringSliceVar(&modes, "mode", []string{"all"}, "启动模式 (all | scheduler | agent | executor)")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(endpoint.Cmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startServer() {
	var app *ioc.App

	// 核心：基于模式选择【专门】的注入器
	// scheduler 模式下不创建 Executor，executor 模式下不创建 DB/Redis
	if len(modes) == 1 {
		switch modes[0] {
		case ioc.ModeScheduler:
			app = ioc.InitSchedulerApp()
		case ioc.ModeExecutor:
			app = ioc.InitExecutorApp()
		case ioc.ModeAgent:
			app = ioc.InitAgentApp()
		default:
			app = ioc.InitApp()
		}
	} else {
		// 组合模式使用全量注入
		app = ioc.InitApp()
	}

	// 此时 app 中的某些字段（如 app.Server 或 app.Executor）可能为 nil
	// GetServers 已经做了 nil 检查，不会导致 panic
	servers := app.GetServers(modes)
	app.StartBackgroundTasks(context.Background(), modes)

	if err := ego.New().Serve(servers...).Run(); err != nil {
		elog.Panic("app_run_error", elog.FieldErr(err))
	}
}

func initViper() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.WatchConfig()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: 配置文件读取失败: %v\n", err)
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}
}
