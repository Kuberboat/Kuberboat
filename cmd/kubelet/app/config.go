package app

import (
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
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
		glog.Fatal(err)
	}

	err = yaml.Unmarshal(cfgFile, cfg)
	if err != nil {
		glog.Fatal(err)
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		glog.Fatalf("invalid port number: %v", cfg.Port)
	}

	return cfg
}
