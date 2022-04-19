package main

import (
	"flag"
	"fmt"
	"os"

	"p9t.io/kuberboat/cmd/kubelet/app"
)

var (
	// configPath is the path to configuration file.
	configPath string
)

func init() {
	homePath, _ := os.UserHomeDir()
	flag.Set("logtostderr", "true")
	flag.StringVar(&configPath, "config", fmt.Sprintf("%s/.kube/kubelet_config.yaml", homePath), "set path to the kubelet configuration file")
}

func main() {
	flag.Parse()
	config := app.BuildConfig(configPath)
	app.StartServer(config)
}
