package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"p9t.io/kuberboat/pkg/kubectl/client"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs JOBNAME",
	Short: "Print the output of a Cuda Job",
	Long: `Print the output of a Cuda Job

Examples:
  # get output of cuda job`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		getLog(args[0])
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}

func getLog(jobName string) {
	client := client.NewCtlClient()
	resp, err := client.GetJobLog(jobName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(resp.Log)
}
