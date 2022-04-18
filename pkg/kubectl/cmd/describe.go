/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific resource or group of resources.",
	Long: `Show details of a specific resource or group of resources.

Examples:
  # Describe a pod
  kubectl describe pod podName1 podName2

  # Describe all pods
  kubectl describe pods`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		switch resourceType {
		case "pod":
			describePods(args[1:])
		case "pods":
			describePods(nil)
		default:
			log.Fatalf("%v is not a supported resource type", resourceType)
		}
		fmt.Println("describe called")
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// describeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// describeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func describePods(podNames []string) {
	// client := client.NewCtlClient()
	if podNames == nil {
		// TODO: describe all the resources
	} else {
		// TODO: describe specified resources
	}
}
