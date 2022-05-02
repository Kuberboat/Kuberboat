package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

const (
	PrometheusAddress    string        = "http://localhost:9090"
	QueryTimeout         time.Duration = 8 * time.Second
	MonitorInterval      time.Duration = 10 * time.Second
	UsageComputeDuration string        = "20s"
)

// MetricsManager monitors the CPU and memory usage of all the ready pods at set intervals.
type MetricsManager interface {
	// StartMonitor monitors all the ready pods at set intervals.
	StartMonitor()
}

type metricsManagerInner struct {
	prometheusAPI    v1.API
	componentManager apiserver.ComponentManager
}

func NewMetricsManager(componentManager apiserver.ComponentManager) (MetricsManager, error) {
	client, err := api.NewClient(api.Config{
		Address: PrometheusAddress,
	})
	if err != nil {
		return nil, err
	}
	return &metricsManagerInner{
		prometheusAPI:    v1.NewAPI(client),
		componentManager: componentManager,
	}, nil
}

func (mm *metricsManagerInner) StartMonitor() {
	for range time.Tick(MonitorInterval) {
		for _, pod := range mm.componentManager.ListPodsByPhase(core.PodReady) {
			go mm.podCPUUsage(pod)
			go mm.podMemoryUsage(pod)
		}
	}
}

// podCPUUsage queries the average CPU usage of a given pod in certain seconds in the past.
func (mm *metricsManagerInner) podCPUUsage(pod *core.Pod) (float64, error) {
	var queryBuilder strings.Builder

	// Pause container
	pauseName := core.GetPodSpecificPauseName(pod)
	containerQuery := containerCPUUsageQuery(pauseName)
	queryBuilder.WriteString(containerQuery)

	// Other containers
	for _, container := range pod.Spec.Containers {
		containerName := core.GetPodSpecificName(pod, container.Name)
		containerQuery = containerCPUUsageQuery(containerName)
		queryBuilder.WriteString(" or ")
		queryBuilder.WriteString(containerQuery)
	}

	// Sum the results
	query := "sum(" + queryBuilder.String() + ")"

	// Query Promethus
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result, warnings, err := mm.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		glog.Errorf("fail to get cpu usage from prometheus: %v\n", err)
		return 0.0, err
	}
	if len(warnings) > 0 {
		glog.Warningf("warnings from prometheus: %v\n", warnings)
	}
	if result.(model.Vector).Len() == 0 {
		returnErr := fmt.Errorf("fail to get cpu usage for pod %s: no data from prometheus", pod.Name)
		glog.Errorln(returnErr)
		return 0.0, returnErr
	}

	glog.Infof("pod %s cpu usage: %f\n", pod.Name, float64(result.(model.Vector)[0].Value))
	return float64(result.(model.Vector)[0].Value), nil
}

// podMemoryUsage queries the average memory usage of a given pod in certain seconds in the past.
func (mm *metricsManagerInner) podMemoryUsage(pod *core.Pod) (uint64, error) {
	var queryBuilder strings.Builder

	// Pause container
	pauseName := core.GetPodSpecificPauseName(pod)
	containerQuery := containerMemoryUsageQuery(pauseName)
	queryBuilder.WriteString(containerQuery)

	// Other containers
	for _, container := range pod.Spec.Containers {
		containerName := core.GetPodSpecificName(pod, container.Name)
		containerQuery = containerMemoryUsageQuery(containerName)
		queryBuilder.WriteString(" or ")
		queryBuilder.WriteString(containerQuery)
	}

	// Sum the results
	query := "sum(" + queryBuilder.String() + ")"

	// Query Promethus
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result, warnings, err := mm.prometheusAPI.Query(ctx, query, time.Now())
	if err != nil {
		glog.Errorf("fail to get memory usage from prometheus: %v\n", err)
		return 0, err
	}
	if len(warnings) > 0 {
		glog.Warningf("warnings from prometheus: %v\n", warnings)
	}
	if result.(model.Vector).Len() == 0 {
		returnErr := fmt.Errorf("fail to get memory usage for pod %s", pod.Name)
		glog.Errorln(returnErr)
		return 0, returnErr
	}

	glog.Infof("pod %s memory usage: %d bytes\n", pod.Name, uint64(result.(model.Vector)[0].Value))
	return uint64(result.(model.Vector)[0].Value), nil
}

// containerCPUUsageQuery is a helper function that generates the PromQL to query a container's
// CPU usage.
func containerCPUUsageQuery(containerName string) string {
	var query strings.Builder
	query.WriteString("sum(rate(container_cpu_usage_seconds_total{name=\"")
	query.WriteString(containerName)
	query.WriteString("\"}[")
	query.WriteString(UsageComputeDuration)
	query.WriteString("])) by (name)")
	return query.String()
}

// containerMemoryUsageQuery is a helper function that generates the PromQL to query a container's
// memory usage.
func containerMemoryUsageQuery(containerName string) string {
	var query strings.Builder
	query.WriteString("avg_over_time(container_memory_usage_bytes{name=\"")
	query.WriteString(containerName)
	query.WriteString("\"}[")
	query.WriteString(UsageComputeDuration)
	query.WriteString("])")
	return query.String()
}

// GeneratePrometheusTargets writes all the endpoints that prometheus needs to listen on into a config file.
// Prometheus will check this file regularly to get the latest info.
func GeneratePrometheusTargets(nodes []*core.Node) error {
	// Create the directory of prometheus target file if not exists
	rootPath, err := os.Getwd()
	if err != nil {
		return err
	}
	dirName := filepath.Join(rootPath, core.PROMETHEUS_TARGET_DIR)
	err = os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		return err
	}

	// Construct file contents
	targets := make([]core.PrometheusTargetObject, 0, len(nodes))
	for _, node := range nodes {
		targetAddr := fmt.Sprintf("%s:%d", node.Status.Address, core.CADVISOR_PORT)
		targetObj := core.PrometheusTargetObject{
			Targets: []string{targetAddr},
			Label: core.PrometheusTargetLabel{
				Job: node.Name,
			},
		}
		targets = append(targets, targetObj)
	}
	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return err
	}

	// Write to prometheus target file
	fileName := filepath.Join(dirName, core.PROMETHEUS_TARGET_FILE)
	err = os.WriteFile(fileName, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
