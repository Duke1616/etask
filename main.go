package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Duke1616/etask/cmd/endpoint"
	"github.com/Duke1616/etask/ioc"
	"github.com/fsnotify/fsnotify"
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
	defaultCfg := dir + "/config/config.yaml"
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
	// 1. 初始化所有模块共享的基础设施（仅连接，不启动业务）
	base := ioc.InitBase()
	app := &ioc.App{Base: base}

	// 2. 动态装配模式
	modeMap := make(map[string]bool)
	for _, m := range modes {
		modeMap[m] = true
	}

	// 始终尝试加载 Web 模块（提供健康检查和管理 API）
	app.Load(ioc.InitWebModule(base))

	// 根据具体模式“物理激活”相应模块
	if modeMap[ioc.ModeAll] || modeMap[ioc.ModeScheduler] {
		app.Load(ioc.InitSchedulerModule(base))
		app.Load(ioc.InitSchedulerServerModule(base))
	}

	if modeMap[ioc.ModeAll] || modeMap[ioc.ModeExecutor] {
		app.Load(ioc.InitExecutorModule(base))
	}

	if modeMap[ioc.ModeAll] || modeMap[ioc.ModeAgent] {
		app.Load(ioc.InitAgentModule(base))
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
		setLogLevel()
	}

	// 监听配置变更，支持动态切换日志级别
	viper.OnConfigChange(func(in fsnotify.Event) {
		setLogLevel()
	})
}

// setLogLevel 根据配置文件中的 log.debug 动态调整全局日志级别
func setLogLevel() {
	if viper.GetBool("log.debug") {
		elog.DefaultLogger.SetLevel(elog.DebugLevel)
		elog.DefaultLogger.Debug("已根据配置开启 Debug 日志级别")
	} else {
		elog.DefaultLogger.SetLevel(elog.InfoLevel)
	}
}
