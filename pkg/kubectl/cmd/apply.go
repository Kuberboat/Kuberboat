package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	valid "github.com/asaskevich/govalidator"
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
			case string(core.DNSType):
				applyDNS(data)
			case string(core.JobType):
				applyJob(data)
			case string(core.AutoscalerType):
				applyAutoscaler(data)
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
	client := client.NewCtlClient()
	response, err := client.CreateService(&service)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Service created\n", response.Status)
}

func applyDNS(data []byte) {
	var dns core.DNS
	if err := yaml.Unmarshal(data, &dns); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	// Do some sanity checks for dns.
	if len(dns.Name) == 0 {
		log.Fatalf("name not specified")
	}
	if !valid.IsDNSName(dns.Spec.Host) {
		log.Fatalf("host is not a valid domain name")
	}
	for _, mapping := range dns.Spec.Paths {
		if len(mapping.Path) == 0 {
			log.Fatalf("path not specified")
		}
		if mapping.Path[0] != '/' || strings.Contains(mapping.Path, " ") || strings.HasSuffix(mapping.Path, "/") {
			log.Fatalf("invalid path")
		}
		if len(mapping.ServiceName) == 0 {
			log.Fatalf("service name not specified")
		}
		if mapping.ServicePort == 0 {
			log.Fatalf("service port not specified")
		}
	}
	client := client.NewCtlClient()
	response, err := client.CreateDNS(&dns)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;DNS created\n", response.Status)
}

func applyJob(data []byte) {
	var job core.Job
	if err := yaml.Unmarshal(data, &job); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	cudaFile, err := os.ReadFile(job.CudaPath)
	if err != nil {
		log.Fatal(err)
	}
	job.CudaData = cudaFile
	scriptFile, err := os.ReadFile(job.ScriptPath)
	if err != nil {
		log.Fatal(err)
	}
	job.ScriptData = scriptFile
	client := client.NewCtlClient()
	response, err := client.CreateJob(&job)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Job created\n", response.Status)
}

func applyAutoscaler(data []byte) {
	var autoscaler core.HorizontalPodAutoscaler
	if err := yaml.Unmarshal(data, &autoscaler); err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	if autoscaler.Spec.ScaleTargetRef.Kind != core.DeploymentType {
		log.Fatalf("target object must be deployment")
	}
	if autoscaler.Spec.MinReplicas > autoscaler.Spec.MaxReplicas {
		log.Fatalf("number of min replicas must be no greater than that of max replicas")
	}
	if autoscaler.Spec.ScaleInterval < 12 {
		log.Fatalf("scale interval cannot be less than 12s")
	}
	for _, metric := range autoscaler.Spec.Metrics {
		if metric.Resource != core.ResourceCPU && metric.Resource != core.ResourceMemory {
			log.Fatalf("unsupported metric: %s", metric.Resource)
		}
	}
	client := client.NewCtlClient()
	response, err := client.CreateAutoscaler(&autoscaler)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response status: %v ;Autoscaler created\n", response.Status)
}
