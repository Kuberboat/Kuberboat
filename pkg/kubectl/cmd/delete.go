/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"p9t.io/kuberboat/pkg/kubectl/client"
)

// deleteCmd represents the delete command
var (
	all       bool
	deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete resources by file names, stdin, resources and names, or by resources and label selector.",
		Long: `Examples:
  # Delete a pod using the name
  kubectl delete pod podName

  # Delete specified pods
  kubectl delete pods podName1 podName2 ...

  # Delete all pods
  kubectl delete pods --all`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			switch resourceType {
			case "pod":
				deletePods([]string{args[1]})
			case "pods":
				if all {
					deletePods(nil)
				} else {
					deletePods(args[1:])
				}
			default:
				log.Fatalf("%v is not supported\n", resourceType)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	deleteCmd.Flags().BoolVar(&all, "all", false, "weather delete all the specified resources")
}

func deletePods(podNames []string) {
	client := client.NewCtlClient()
	if podNames == nil {
		response, err := client.DeletePod("")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Reponse status: %v ;Pods deleted\n", response.Status)
	} else {
		for _, name := range podNames {
			response, err := client.DeletePod(name)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Response status: %v ;Pod %v deleted\n", response.Status, name)
		}
	}
}
