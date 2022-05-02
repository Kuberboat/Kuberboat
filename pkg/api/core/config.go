package core

const APISERVER_PORT uint16 = 6443
const CADVISOR_PORT uint16 = 8080
const PROMETHEUS_TARGET_DIR string = "out/config"
const PROMETHEUS_TARGET_FILE string = "prom_targets.json"

type ConfigKind struct {
	Kind string
}

type PrometheusTargetObject struct {
	Targets []string              `json:"targets"`
	Label   PrometheusTargetLabel `json:"labels"`
}

type PrometheusTargetLabel struct {
	Job string `json:"job"`
}
