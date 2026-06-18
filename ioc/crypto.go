package ioc

import (
	"github.com/Duke1616/ecmdb/pkg/cryptox"
	"github.com/spf13/viper"
)

// InitCrypto 初始化执行单元变量加解密组件。
func InitCrypto() cryptox.Crypto {
	type Config struct {
		Version string `mapstructure:"version"`
		Key     string `mapstructure:"key"`
	}

	var cfg Config
	if err := viper.UnmarshalKey("encryption", &cfg); err != nil {
		panic(err)
	}
	if cfg.Version == "" {
		cfg.Version = "V1"
	}
	if cfg.Key == "" {
		cfg.Key = "1234567890"
	}

	return cryptox.NewCryptoManager("V2").
		Register("V2", cryptox.MustNewAESCryptoV2(cfg.Key)).
		Register(cfg.Version, cryptox.MustNewAESCrypto(cfg.Key)).
		WithLegacyAlgo(cfg.Version)
}
