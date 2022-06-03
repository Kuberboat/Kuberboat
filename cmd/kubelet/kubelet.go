package main

import (
	"flag"

	"p9t.io/kuberboat/cmd/kubelet/app"
)

var (
	// dnsIP is the IP address of CoreDNS name server.
	dnsIP string
)

func init() {
	flag.Set("logtostderr", "true")
}

func main() {
	flag.Parse()
	app.StartServer()
}
