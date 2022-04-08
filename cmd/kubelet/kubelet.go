package main

import (
	"flag"

	"p9t.io/kuberboat/cmd/kubelet/app"
)

var (
	// configPath is the path to configuration file.
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "~/.kube/config.yml", "set path to the configuration file")
}

func main() {
	flag.Parse()
	config := app.BuildConfig(configPath)
	app.StartServer(config)
}
