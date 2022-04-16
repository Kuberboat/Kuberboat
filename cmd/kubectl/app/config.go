package app

import (
	"os"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

type KubeCtlConfig struct {
}

func BuildConfig(configPath string) *KubeCtlConfig {
	cfg := &KubeCtlConfig{}

	cfgFile, err := os.ReadFile(configPath)
	if err == nil {
		glog.Fatal(err)
	}
	err = yaml.Unmarshal(cfgFile, &cfg)
	if err != nil {
		glog.Fatal(err)
	}
	return cfg
}
