/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubectl/client"
)

// rootCmd represents the base command when called without any subcommands
var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "kubectl",
		Short: "kubectl controls the Kubernetes cluster manager.",
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	log.SetPrefix("error: ")
	log.SetFlags(0)

	cobra.OnInitialize(buildConfig)

	homePath, _ := os.UserHomeDir()
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", fmt.Sprintf("%s/.kube/kubectl_config.yaml", homePath), "config file")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// rootCmd.PersistentFlags().
}

var config core.Config
var context2cluster map[string]*core.ClusterWithName

func buildConfig() {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Panic(err)
	}
	var configKind core.ConfigKind
	err = yaml.Unmarshal(data, &configKind)
	if err != nil {
		log.Fatal("error decoding your config type")
	}
	if configKind.Kind != "Config" {
		log.Fatalf("Cannot use %v config type to config kubectl", configKind.Kind)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	context2cluster = make(map[string]*core.ClusterWithName)
	for _, context := range config.Contexts {
		found := false
		for i := 0; i < len(config.Clusters); i++ {
			if config.Clusters[i].Name == context.Context {
				context2cluster[context.Name] = &config.Clusters[i]
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("cluster %v not found", context.Context)
		}
	}
	if config.Duration != 0 {
		client.CONN_TIMEOUT = time.Second * time.Duration(config.Duration)
	}
	if config.CurrentContext.Context != "" {
		cluster := context2cluster[config.CurrentContext.Name]
		if cluster == nil {
			log.Fatalf("cluster %v not found", config.CurrentContext.Context)
		}
		if cluster.Server != "" {
			client.APISERVER_URL = cluster.Server
		}
		if cluster.Port != 0 {
			client.APISERVER_PORT = cluster.Port
		}
	}
}
