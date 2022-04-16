package main

import (
	"flag"

	"p9t.io/kuberboat/cmd/apiserver/app"
)

func main() {
	flag.Parse()
	app.StartServer()
}
