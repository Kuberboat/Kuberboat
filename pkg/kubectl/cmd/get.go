/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var (
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources.",
		Long: `Prints a table of the most important information about the specified resources. You can filter the list using a label
selector and the --selector flag. If the desired resource type is namespaced you will only see results in your current
namespace unless you pass --all-namespaces.

Uninitialized objects are not shown unless --include-uninitialized is passed.
	
By specifying the output as 'template' and providing a Go template as the value of the --template flag, you can filter
the attributes of the fetched resources.
	
Use "kubectl api-resources" for a complete list of supported resources.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resourceType := args[0]
			switch resourceType {
			case "pod":
				return getPods(args[1:])
			case "pods":
				return getAllPods()
			default:
				return fmt.Errorf("%q is not a supported resource type", resourceType)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getPods(podNames []string) error {
	fmt.Println("get these", podNames, "pods")
	return nil
}

func getAllPods() error {
	fmt.Println("get all pods")
	return nil
}
