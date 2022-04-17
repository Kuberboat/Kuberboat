/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubectl/client"
)

// applyCmd represents the apply command
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
				glog.Errorf("%w", err)
			}
			var configKind core.ConfigKind
			err = yaml.Unmarshal(data, &configKind)
			if err != nil {
				glog.Error("error decoding your type")
			}
			switch configKind.Kind {
			case "Pod":
				applyPod(data)
			default:
				glog.Errorf("%v is not supported", configKind.Kind)
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
		glog.Fatalf("cannot unmarshal data: %v", err)
	}
	glog.Infof("pod configuration got is %#v\n", pod)
	client := client.NewCtlClient()
	response, err := client.CreatePod(&pod)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Response status: %v ;Pod created", response.Status)
}
