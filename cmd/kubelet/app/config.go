package app

import (
	"io/ioutil"

	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
)

// KubeletConfig holds configuration options for kubelet.
type KubeletConfig struct {
	// Port number to start grpc server.
	Port uint16 `yaml:"port"`
}

// BuildConfig builds kubelet config from command line arguments.
func BuildConfig(configPath string) *KubeletConfig {
	cfg := new(KubeletConfig)

	cfgFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Warningf("error opening config file, using defalt config: %v", err.Error())
		return &KubeletConfig{
			Port: 4000,
		}
	}

	err = yaml.Unmarshal(cfgFile, cfg)
	if err != nil {
		glog.Fatal(err)
	}

	if cfg.Port > 65535 {
		glog.Fatalf("invalid port number: %v", cfg.Port)
	}

	return cfg
}
