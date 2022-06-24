package main

import (
	"github.com/fmgornick/dynamic-envoy/config/utils/parser"
	"github.com/fmgornick/dynamic-envoy/config/utils/print"
	"github.com/fmgornick/dynamic-envoy/config/utils/usercfg"
)

const xdsPort = 6969

func main() {
	// p := processor.NewProcessor()
	// err := p.ProcessDir("bags2")
	// if err != nil {
	// 	panic(err)
	// }

	// go func() {
	// 	server := server.NewServer(context.Background(), p.Cache, &test.Callbacks{})
	// 	xdsServer.RunServer(context.Background(), server, xdsPort)
	// }()

	// print.PrettyPrint(p.Resource)
	// for {
	// }
	bags, err := usercfg.ParseDir("../databags/target")
	if err != nil {
		panic(err)
	}
	config, err := parser.Parse(bags)
	print.PrettyPrint(&config)
}
