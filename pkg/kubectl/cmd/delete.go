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
  kubectl delete pod <podName>

  # Delete specified pods
  kubectl delete pods <podName1> <podName2> ...

  # Delete all pods
  kubectl delete pods --all
  
  # Delete a service using the name
  kubectl delete service <serviceName>

  # Delete specified services
  kubectl delete services <serviceName1> <serviceName2> ...
  
  # Delete all services
  kubectl delete services --all
  
  # Delete a deployment using the name
  kubectl delete deployment <deploymentName>

  # Delete specified deployments
  kubectl delete deployments <deploymentName1> <deploymentName2> ...
  
  # Delete all deployments
  kubectl delete deployments --all`,
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
			case "service":
				deleteServices([]string{args[1]})
			case "services":
				if all {
					deleteServices(nil)
				} else {
					deleteServices(args[1:])
				}
			case "deployment":
				deleteDeployments([]string{args[1]})
			case "deployments":
				if all {
					deleteDeployments(nil)
				} else {
					deleteDeployments(args[1:])
				}
			default:
				log.Fatalf("%v is not supported\n", resourceType)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVar(&all, "all", false, "weather delete all the specified resources")
}

func deletePods(podNames []string) {
	client := client.NewCtlClient()
	if podNames == nil {
		response, err := client.DeletePod("")
		if err != nil {
			log.Print(err)
		} else {
			fmt.Printf("Reponse status: %v ;Pods deleted\n", response.Status)
		}
	} else {
		for _, name := range podNames {
			response, err := client.DeletePod(name)
			if err != nil {
				log.Print(err)
			} else {
				fmt.Printf("Response status: %v ;Pod %v deleted\n", response.Status, name)
			}
		}
	}
}

func deleteServices(serviceNames []string) {
	client := client.NewCtlClient()
	if serviceNames == nil {
		response, err := client.DeleteService("")
		if err != nil {
			log.Print(err)
		} else {
			fmt.Printf("Reponse status: %v ;Services deleted\n", response.Status)
		}
	} else {
		for _, name := range serviceNames {
			response, err := client.DeleteService(name)
			if err != nil {
				log.Print(err)
			} else {
				fmt.Printf("Response status: %v ;Service %v deleted\n", response.Status, name)
			}
		}
	}
}

func deleteDeployments(deploymentNames []string) {
	client := client.NewCtlClient()
	if deploymentNames == nil {
		response, err := client.DeleteDeployment("")
		if err != nil {
			log.Print(err)
		} else {
			fmt.Printf("Reponse status: %v ;Deployments deleted\n", response.Status)
		}
	} else {
		for _, name := range deploymentNames {
			response, err := client.DeleteDeployment(name)
			if err != nil {
				log.Print(err)
			} else {
				fmt.Printf("Response status: %v ;Deployment %v deleted\n", response.Status, name)
			}
		}
	}
}
