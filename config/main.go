package main

import (
	"context"

	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"
	parser "github.com/fmgornick/dynamic-envoy/config/utils/parser"
	processor "github.com/fmgornick/dynamic-envoy/config/utils/processor"
	usercfg "github.com/fmgornick/dynamic-envoy/config/utils/usercfg"
	xdsServer "github.com/fmgornick/dynamic-envoy/config/utils/xdsServer"
)

const xdsPort = 6969

func main() {
	bags, err := usercfg.ParseFile("../databags/local/ex1.json")
	if err != nil {
		panic(err)
	}
	config, err := parser.Parse(bags)
	if err != nil {
		panic(err)
	}
	cache, err := processor.Process(config)
	if err != nil {
		panic(err)
	}

	server := server.NewServer(context.Background(), *cache, &test.Callbacks{})
	go func() {
		xdsServer.RunServer(context.Background(), server, xdsPort)
	}()

	for {
	}
}
