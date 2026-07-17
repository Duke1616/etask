package ioc

import (
	artifactSvc "github.com/Duke1616/etask/internal/service/artifact"
	"github.com/Duke1616/etask/pkg/blobstore"
	config "github.com/Duke1616/etask/pkg/config"
)

func InitArtifactConfig() artifactSvc.Config {
	var cfg artifactSvc.Config
	if err := config.UnmarshalKey("artifact", &cfg); err != nil {
		panic(err)
	}
	return cfg
}

func InitArtifactStore(cfg artifactSvc.Config) blobstore.Store {
	store, err := blobstore.New(cfg.Storage)
	if err != nil {
		panic(err)
	}
	return store
}
