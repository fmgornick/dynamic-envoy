package main

import (
	"context"
	"flag"
	"fmt"

	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	prnt "github.com/fmgornick/dynamic-envoy/utils/print"
	processor "github.com/fmgornick/dynamic-envoy/utils/processor"
	watcher "github.com/fmgornick/dynamic-envoy/utils/watcher"
	xdsServer "github.com/fmgornick/dynamic-envoy/utils/xdsServer"
)

var (
	directory string
	nodeId    string
	xdsPort   uint
)

var change chan watcher.Message     // used to keep track of changes to specified directory
var envoy *processor.EnvoyProcessor // used to send new configuration to envoy

// TODO: ADD README.MD
func init() {
	// initialize environment variables, these can be set by user when running program via setting the flags
	flag.StringVar(&directory, "directory", "databags/local", "path to folder containing databag files")
	flag.StringVar(&nodeId, "nodeId", "envoy-instance", "node id of envoy instance")
	flag.UintVar(&xdsPort, "port", 6969, "port number our xds management server is running on")

	// initialize directory watcher and processor
	change = make(chan watcher.Message)
	envoy = processor.NewProcessor(nodeId)
}

func main() {
	// call to take in command line input
	flag.Parse()
	// remove leading "./"
	if directory[:2] == "./" {
		directory = directory[2:]
	}

	// send existing databag files to envoy
	err := envoy.Process(watcher.Message{
		Operation: watcher.Create,
		Path:      directory,
	})
	if err != nil {
		err = fmt.Errorf("error processing config: %+v\n", err)
		panic(err)
	}
	prnt.EnvoyPrint(envoy.Configs)

	// watch for file changes in specified directory
	go func() {
		watcher.Watch(directory, change)
	}()

	// run xds server to send cache updates
	go func() {
		server := server.NewServer(context.Background(), envoy.Cache, &test.Callbacks{})
		xdsServer.RunServer(context.Background(), server, xdsPort)
	}()

	// listen on directory for updates
	// when change is made, process the change and send new snapshot
	for {
		select {
		case msg := <-change:
			// fmt.Printf("processing new change...\n")
			err := envoy.Process(msg)
			if err != nil {
				err = fmt.Errorf("error processing new config: %+v\n", err)
				panic(err)
			}
			prnt.EnvoyPrint(envoy.Configs)
			// fmt.Printf("changes added!!!\n\n")
		}
	}
}
