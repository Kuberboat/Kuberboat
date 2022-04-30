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
		case "deployment":
			describeDeployments(args[1:])
		case "deployments":
			describeDeployments(nil)
		default:
			log.Fatalf("%v is not a supported resource type", resourceType)
		}
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

func describeDeployments(deploymentNames []string) {
	type DisplayedDeployment struct {
		Deployment *core.Deployment
		Pods       []string
	}
	client := client.NewCtlClient()
	var resp *pb.DescribeDeploymentsResponse
	var err error
	if deploymentNames == nil {
		resp, err = client.DescribeDeployments(true, nil)
	} else {
		resp, err = client.DescribeDeployments(false, deploymentNames)
	}

	if err != nil {
		log.Fatal(err)
	}

	var foundDeployments []*core.Deployment
	var displayedDeployments []DisplayedDeployment
	var deploymentPods [][]string
	var notFoundDeployments []string
	err = json.Unmarshal(resp.Deployments, &foundDeployments)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(resp.DeploymentPodNames, &deploymentPods)
	if err != nil {
		log.Fatal(err)
	}
	for index, deployment := range foundDeployments {
		displayedDeployments = append(displayedDeployments, DisplayedDeployment{
			Deployment: deployment,
			Pods:       deploymentPods[index],
		})
	}
	prettyjson, err := json.MarshalIndent(displayedDeployments, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(prettyjson))

	if resp.Status == -2 {
		err = json.Unmarshal(resp.NotFoundDeployments, &notFoundDeployments)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The following deployments are not found: %v\n", notFoundDeployments)
	}
}
