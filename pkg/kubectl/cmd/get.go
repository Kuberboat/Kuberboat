/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubectl/client"
	pb "p9t.io/kuberboat/pkg/proto"
)

// getCmd represents the get command
var (
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources.",
		Long: `Display one or many resources.

Examples:
  # List all pods in ps output format
  kubectl get pods

  # List pods specified by their names
  kubectl get pod podName1 podName2 ...`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			switch resourceType {
			case "pod":
				getPods(args[1:])
			case "pods":
				getPods(nil)
			default:
				log.Fatalf("%v is not a supported resource type", resourceType)
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

func getPods(podNames []string) {
	client := client.NewCtlClient()
	var resp *pb.GetPodsResponse
	var err error
	if podNames == nil {
		resp, err = client.GetPods(true, nil)
	} else {
		resp, err = client.GetPods(false, podNames)
	}

	if err != nil {
		log.Fatal(err)
	}

	var foundPods []*core.Pod
	var notFoundPods []string
	err = json.Unmarshal(resp.Pods, &foundPods)
	if err != nil {
		log.Fatal(err)
	}

	// We might use some other serialization methods here.
	prettyjson, err := json.MarshalIndent(foundPods, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(prettyjson))

	if resp.Status == -2 {
		err = json.Unmarshal(resp.NotFoundPods, &notFoundPods)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The following pods are not found: %v\n", notFoundPods)
	}
}
