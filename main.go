package main

import (
	"context"
	"flag"
	"fmt"

	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	prnt "github.com/fmgornick/dynamic-proxy/app/print"
	processor "github.com/fmgornick/dynamic-proxy/app/processor"
	watcher "github.com/fmgornick/dynamic-proxy/app/watcher"
	xdsServer "github.com/fmgornick/dynamic-proxy/app/xdsServer"
)

var (
	addHttp   bool
	directory string

	iAddr string
	eAddr string
	iPort uint
	ePort uint
)

var change chan watcher.Message     // used to keep track of changes to specified directory
var envoy *processor.EnvoyProcessor // used to send new configuration to envoy

func init() {
	// initialize environment variables, these can be set by user when running program via setting the flags
	flag.BoolVar(&addHttp, "add-http", false, "optional flag for setting up listeners with HTTP compatability")
	flag.StringVar(&directory, "dir", "databags/local", "path to folder containing databag files")

	flag.StringVar(&iAddr, "ia", "127.0.0.1", "address the proxy's internal listener listens on")
	flag.StringVar(&eAddr, "ea", "127.0.0.1", "address the proxy's external listener listens on")
	flag.UintVar(&iPort, "ip", 7777, "port number our internal listener listens on")
	flag.UintVar(&ePort, "ep", 8888, "port number our external listener listens on")

	// initialize directory watcher
	change = make(chan watcher.Message)
}

func main() {
	// call to take in command line input
	flag.Parse()
	envoy = processor.NewProcessor("envoy-instance", addHttp, iAddr, eAddr, iPort, ePort)
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
	snapshot, _ := envoy.Cache.GetSnapshot("envoy-instance")
	prnt.PrettyPrint(snapshot)

	// watch for file changes in specified directory
	go func() {
		watcher.Watch(directory, change)
	}()

	// run xds server to send cache updates
	go func() {
		server := server.NewServer(context.Background(), envoy.Cache, &test.Callbacks{})
		xdsServer.RunServer(context.Background(), server, 6515)
	}()

	// listen on directory for updates
	// when change is made, process the change and send new snapshot
	for {
		select {
		case msg := <-change:
			err := envoy.Process(msg)
			if err != nil {
				err = fmt.Errorf("error processing new config: %+v\n", err)
				panic(err)
			}
			prnt.EnvoyPrint(envoy.Configs)
		}
	}
}
