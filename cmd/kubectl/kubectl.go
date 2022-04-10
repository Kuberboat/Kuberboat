/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"flag"

	"p9t.io/kuberboat/pkg/kubectl/cmd"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "~/.kube/kubectl_config.yaml", "set path to the kubectl configuration file")
	flag.Set("stderrthreshold", "INFO")
}

func main() {
	flag.Parse()
	cmd.Execute()
}
