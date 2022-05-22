package api

const (
	// IP address of the apiserver. Used to inform workers of apiserver's address.
	ApiServerIP = "KUBE_SERVER_IP"
	// Whether kubelet is running on ci. Host DNS feature will cut off ci's connection,
	// so that feature must be disabled on ci.
	CiMode = "KUBE_CI_MODE"
)
