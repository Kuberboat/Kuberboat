package main

import (
	"flag"

	"p9t.io/kuberboat/cmd/apiserver/app"
)

func init() {
	flag.Set("logtostderr", "true")
}

func main() {
	flag.Parse()
	app.StartServer()
}
