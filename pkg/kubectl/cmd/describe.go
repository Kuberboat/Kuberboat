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
  kubectl describe pods

  # Describe a service
  kubectl describe service serviceName1 serviceName2
  
  # Describe all services
  kubectl describe services
  
  # Describe a deployment
  kubectl describe deployment deploymentName1 deploymentName2
  
  # Describe all deployments
  kubectl describe deployments,

  # Describe a dns configuration
  kubectl describe dns dnsName1 dnsName2

  # Describe all dns configurations
  kubectl describe dnss`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		switch resourceType {
		case "pod":
			describePods(args[1:])
		case "pods":
			describePods(nil)
		case "service":
			describeServices(args[1:])
		case "services":
			describeServices(nil)
		case "deployment":
			describeDeployments(args[1:])
		case "deployments":
			describeDeployments(nil)
		case "dns":
			describeDNSs(args[1:])
		case "dnss":
			describeDNSs(nil)
		default:
			log.Fatalf("%v is not a supported resource type", resourceType)
		}
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func describePods(podNames []string) {
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

func describeServices(serviceNames []string) {
	type DisplayedServices struct {
		Service *core.Service
		Pods    []string
	}
	client := client.NewCtlClient()
	var resp *pb.DescribeServicesResponse
	var err error
	if serviceNames == nil {
		resp, err = client.DescribeServices(true, nil)
	} else {
		resp, err = client.DescribeServices(false, serviceNames)
	}

	if err != nil {
		log.Fatal(err)
	}

	var foundServices []*core.Service
	var displayedServices []DisplayedServices
	var servicePods [][]string
	var notFoundServices []string
	err = json.Unmarshal(resp.Services, &foundServices)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(resp.ServicePodNames, &servicePods)
	if err != nil {
		log.Fatal(err)
	}
	for index, service := range foundServices {
		displayedServices = append(displayedServices, DisplayedServices{
			Service: service,
			Pods:    servicePods[index],
		})
	}
	prettyjson, err := json.MarshalIndent(displayedServices, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(prettyjson))

	if resp.Status == -2 {
		err = json.Unmarshal(resp.NotFoundServices, &notFoundServices)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The following services are not found: %v\n", notFoundServices)
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

func describeDNSs(dnsNames []string) {
	client := client.NewCtlClient()
	var resp *pb.DescribeDNSsResponse
	var err error
	if dnsNames == nil {
		resp, err = client.DescribeDNSs(true, nil)
	} else {
		resp, err = client.DescribeDNSs(false, dnsNames)
	}

	if err != nil {
		log.Fatal(err)
	}

	var foundDNSs []*core.DNS
	var notFoundDNSs []string
	err = json.Unmarshal(resp.Dnss, &foundDNSs)
	if err != nil {
		log.Fatal(err)
	}

	prettyjson, err := json.MarshalIndent(foundDNSs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(prettyjson))
	if resp.Status == -2 {
		err = json.Unmarshal(resp.NotFoundDnss, &notFoundDNSs)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("The following pods are not found: %v\n", notFoundDNSs)
	}
}
