package main

import (
	"os"

	"github.com/Duke1616/etask/cmd/execute/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()

	// 创建 ego 应用实例
	egoApp := ego.New()

	// 初始化 Agent 应用
	app := ioc.InitExecuteApp()

	// 启动服务
	if err := egoApp.Serve(
		func() server.Server {
			return app.Server
		}(),
	).Run(); err != nil {
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
