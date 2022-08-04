package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	univcfg "github.com/fmgornick/dynamic-proxy/app/config/universal"
	prnt "github.com/fmgornick/dynamic-proxy/app/print"
	processor "github.com/fmgornick/dynamic-proxy/app/processor"
	watcher "github.com/fmgornick/dynamic-proxy/app/watcher"
	xdsServer "github.com/fmgornick/dynamic-proxy/app/xdsServer"
)

var (
	addHttp   bool
	directory string

	iAddr  string
	eAddr  string
	iPort  uint
	ePort  uint
	iCName string
	eCName string
)

var change chan watcher.Message        // used to keep track of changes to specified directory
var gracefulTermination chan os.Signal // sends last update to envoy to clear everything
var envoy *processor.EnvoyProcessor    // used to send new configuration to envoy

func init() {
	// initialize environment variables, these can be set by user when running program via setting the flags
	flag.BoolVar(&addHttp, "add-http", false, "optional flag for setting up listeners with HTTP compatability")
	flag.StringVar(&directory, "dir", "databags/dev", "path to folder containing databag files")

	flag.StringVar(&iAddr, "ia", "0.0.0.0", "address the proxy's internal listener listens on")
	flag.StringVar(&eAddr, "ea", "0.0.0.0", "address the proxy's external listener listens on")
	flag.UintVar(&iPort, "ip", 7777, "port number our internal listener listens on")
	flag.UintVar(&ePort, "ep", 8888, "port number our external listener listens on")
	flag.StringVar(&iCName, "icn", "localhost", "common name of internal listening address")
	flag.StringVar(&eCName, "ecn", "localhost", "common name of external listening address")

	// initialize directory watcher
	change = make(chan watcher.Message)

	// initialize termination handler
	gracefulTermination = make(chan os.Signal, 1)
	signal.Notify(gracefulTermination, syscall.SIGINT, syscall.SIGTERM)
}

func main() {
	// call to take in command line input
	flag.Parse()
	listenerInfo := univcfg.ListenerInfo{
		InternalAddress:    iAddr,
		ExternalAddress:    eAddr,
		InternalPort:       iPort,
		ExternalPort:       ePort,
		InternalCommonName: iCName,
		ExternalCommonName: eCName,
	}
	envoy = processor.NewProcessor("envoy-instance", addHttp, listenerInfo)
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
	envoy.Cache.GetSnapshot("envoy-instance")
	prnt.EnvoyPrint(envoy.Configs)

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
		case _ = <-gracefulTermination:
			fmt.Printf("\nemptying configuration...\n")
			envoy.ClearConfig()
			prnt.EnvoyPrint(envoy.Configs)
			fmt.Printf("done!!!\n")
			os.Exit(0)
		}
	}
}
