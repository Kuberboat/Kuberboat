/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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

YAML format is accepted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			var configKind ConfigKind
			err = yaml.Unmarshal(data, &configKind)
			if err != nil {
				return fmt.Errorf("error decoding your type")
			}
			switch configKind.Kind {
			case "Pod":
				return applyPod(data)
			default:
				return fmt.Errorf("%v is not supported", configKind.Kind)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&file, "file", "f", "", "specify the configuration file")
	applyCmd.MarkFlagRequired("file")
}

func applyPod(data []byte) error {
	var pod Pod
	err := yaml.Unmarshal(data, &pod)
	if err != nil {
		return fmt.Errorf("cannot unmarshal data: %v", err)
	}
	fmt.Printf("pod configuration got is %#v", pod)
	return nil
}
