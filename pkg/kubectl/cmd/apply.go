/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubectl/client"
)

var (
	file     string
	applyCmd = &cobra.Command{
		Use:   "apply [-f FILENAME]",
		Short: "Apply a configuration to a resource by file name or stdin",
		Long: `Apply a configuration to a resource by file name or stdin. The resource name must be specified. This resource will be
created if it doesn't exist yet. To use 'apply', always create the resource initially with either 'apply' or 'create
--save-config'.

YAML format is accepted.

Examples:
  # Apply the configuration in pod.yaml to a pod
  kubectl apply -f ./pod.yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			data, err := os.ReadFile(file)
			if err != nil {
				log.Fatal(err)
			}
			var configKind core.ConfigKind
			err = yaml.Unmarshal(data, &configKind)
			if err != nil {
				log.Fatal("error decoding your config type")
			}
			switch configKind.Kind {
			case "Pod":
				applyPod(data)
			case "Node":
				applyNode(data)
			default:
				log.Fatalf("%v is not supported", configKind.Kind)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&file, "file", "f", "", "specify the configuration file")
	applyCmd.MarkFlagRequired("file")
}

func applyPod(data []byte) {
	var pod core.Pod
	err := yaml.Unmarshal(data, &pod)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	fmt.Printf("pod configuration got is %+v\n", pod)
	client := client.NewCtlClient()
	response, err := client.CreatePod(&pod)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Pod created\n", response.Status)
}

func applyNode(data []byte) {
	var node core.Node
	err := yaml.Unmarshal(data, &node)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	client := client.NewCtlClient()
	response, err := client.RegisterNode(&node)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Node created\n", response.Status)
}
