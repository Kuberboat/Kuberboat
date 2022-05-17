package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// configCmd represents the config command
var (
	configCmd = &cobra.Command{
		Use:   "config SUBCOMMAND [options]",
		Short: "Modify kubeconfig files using subcommands like \"kubectl config set current-context my-context\"",
		Long: `Modify kubeconfig files using subcommands like "kubectl config set current-context my-context"

Avaliable Commands:
  use-context     Set the current-context in a kubeconfig file`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("config called")
		},
	}
	useContextCmd = &cobra.Command{
		Use:   "use-context",
		Short: "Set the current-context in a kubeconfig file.",
		Long: `Set the current-context in a kubeconfig file.

Examples:
  # Use the context for the minikube cluster
  kubectl config use-context minikube`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			useContext(args[0])
		},
	}
	viewCmd = &cobra.Command{
		Use:   "view",
		Short: "Display merged kubeconfig settings or a specified kubeconfig file.",
		Long: `Display merged kubeconfig settings or a specified kubeconfig file.

Examples:
  # Show merged kubeconfig settings
  kubectl config view`,
		Run: func(cmd *cobra.Command, args []string) {
			showContext()
		},
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(useContextCmd)
	configCmd.AddCommand(viewCmd)
}

func useContext(contextName string) {
	cluster := context2cluster[contextName]
	if cluster == nil {
		log.Fatalf("cluster %v not found", config.CurrentContext.Context)
	}
	config.CurrentContext.Context = cluster.Name
	config.CurrentContext.Name = contextName
	data, err := yaml.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(cfgFile, data, 0644); err != nil {
		log.Panic(err)
	}
}

func showContext() {
	fmt.Printf("Current context: %+v\n", config.CurrentContext)
	fmt.Printf("Avaliable cluster: %+v\n", config.Clusters)
	fmt.Printf("Avaliable context: %+v\n", config.Contexts)
}
