package main

import (
	"os"

	"github.com/Duke1616/etask/ioc"
	"github.com/fsnotify/fsnotify"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()

	// 1. 初始化共享基础设施
	base := ioc.InitBase()
	app := &ioc.App{Base: base}

	// 2. 物理加载 Executor 模块
	app.Load(ioc.InitExecutorModule(base))

	// 3. 获取并启动服务
	servers := app.GetServers([]string{ioc.ModeExecutor})
	egoApp := ego.New()
	if err := egoApp.Serve(servers...).Run(); err != nil {
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

	viper.WatchConfig()
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	setLogLevel()

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
