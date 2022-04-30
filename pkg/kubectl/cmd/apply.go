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
			case string(core.PodType):
				applyPod(data)
			case string(core.NodeType):
				applyNode(data)
			case string(core.DeploymentType):
				applyDeployment(data)
			case string(core.ServiceType):
				applyService(data)
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
	if err := yaml.Unmarshal(data, &pod); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	client := client.NewCtlClient()
	response, err := client.CreatePod(&pod)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Pod created\n", response.Status)
}

func applyNode(data []byte) {
	var node core.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	client := client.NewCtlClient()
	response, err := client.RegisterNode(&node)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Node created\n", response.Status)
}

func applyDeployment(data []byte) {
	var deployment core.Deployment
	if err := yaml.Unmarshal(data, &deployment); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}

	client := client.NewCtlClient()
	response, err := client.CreateDeployment(&deployment)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Deployment created\n", response.Status)
}

func applyService(data []byte) {
	var service core.Service
	if err := yaml.Unmarshal(data, &service); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	// If target port is not specified, it should default to corresponding service-exposed port.
	for i := range service.Spec.Ports {
		if service.Spec.Ports[i].TargetPort == 0 {
			service.Spec.Ports[i].TargetPort = service.Spec.Ports[i].Port
		}
	}
	fmt.Println(service)
	client := client.NewCtlClient()
	response, err := client.CreateService(&service)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Service created\n", response.Status)
}
