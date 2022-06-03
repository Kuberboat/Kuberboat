package main

import (
	"flag"

	"p9t.io/kuberboat/cmd/kubelet/app"
)

var (
	// dnsIP is the IP address of CoreDNS name server.
	dnsIP       string
	etcdServers string
)

func init() {
	flag.Set("logtostderr", "true")
	flag.StringVar(&etcdServers, "etcd-servers", "localhost:2379", "List of etcd servers to connect with (scheme://ip:port), comma separated.")
}

func main() {
	flag.Parse()
	app.StartServer(etcdServers)
}
