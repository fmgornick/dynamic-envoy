package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"
	usercfg "github.com/fmgornick/dynamic-envoy/utils/config/user"
	parser "github.com/fmgornick/dynamic-envoy/utils/parser"
	processor "github.com/fmgornick/dynamic-envoy/utils/processor"
	watcher "github.com/fmgornick/dynamic-envoy/utils/watcher"
	xdsServer "github.com/fmgornick/dynamic-envoy/utils/xdsServer"
)

var (
	directory string
	nodeId    string
	xdsPort   uint
)

var configs map[string]*univcfg.Config   // keeps track of each file/directory configuration
var eProcessor *processor.EnvoyProcessor // used to send new configuration to envoy

func init() {
	// initialize environment variables, these can be set by user when running program via setting the flags
	flag.StringVar(&directory, "directory", "./databags/local", "path to folder containing databag files")
	flag.StringVar(&nodeId, "nodeId", "envoy-instance", "node id of envoy instance")
	flag.UintVar(&xdsPort, "port", 6969, "port number our xds management server is running on")

	// initialize our config map and processor
	configs = make(map[string]*univcfg.Config)
	eProcessor = processor.NewProcessor(nodeId)
}

func main() {
	// call to take in command line input
	flag.Parse()

	// send existing databag files to envoy
	err := ProcessChange(watcher.Message{
		Operation: watcher.Create,
		Path:      directory,
	})
	if err != nil {
		err = fmt.Errorf("error processing config: %+v\n", err)
		panic(err)
	}

	// watch for file changes in specified directory
	change := make(chan watcher.Message)
	go func() {
		watcher.Watch(directory, change)
	}()

	// run xds server to send cache updates
	go func() {
		server := server.NewServer(context.Background(), eProcessor.Cache, &test.Callbacks{})
		xdsServer.RunServer(context.Background(), server, xdsPort)
	}()

	// listen on directory for updates
	// when change is made, process the change and send new snapshot
	for {
		select {
		case msg := <-change:
			fmt.Printf("processing new change...\n")
			err := ProcessChange(msg)
			if err != nil {
				err = fmt.Errorf("error processing new config: %+v\n", err)
				panic(err)
			}
			fmt.Printf("changes added!!!\n\n")
		}
	}
}

// update snapshot and send to server based on change
func ProcessChange(msg watcher.Message) error {
	var bags []usercfg.Bag
	var config *univcfg.Config

	// what to do depending on operation...
	// created new file:   add it's configuration to our existing one
	// file changed:       delete existing configuration of file, then re-add it
	// file moved/deleted: delete existing configuration of file
	switch msg.Operation {
	case watcher.Create:
		// check if changed file is a directory or not
		fileInfo, err := os.Stat(msg.Path)
		if err != nil {
			return fmt.Errorf("path check error: %+v", err)
		}

		if fileInfo.IsDir() {
			bags, err = usercfg.ParseDir(msg.Path)
			if err != nil {
				return err
			}
		} else {
			bags, err = usercfg.ParseFile(msg.Path)
			if err != nil {
				return err
			}
		}
		config, err = parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	case watcher.Modify:
		// check if changed file is a directory or not
		fileInfo, err := os.Stat(msg.Path)
		if err != nil {
			return fmt.Errorf("path check error: %+v", err)
		}

		delete(configs, msg.Path)

		if fileInfo.IsDir() {
			bags, err = usercfg.ParseDir(msg.Path)
			if err != nil {
				return err
			}
		} else {
			bags, err = usercfg.ParseFile(msg.Path)
			if err != nil {
				return err
			}
		}
		config, err = parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	default:
		delete(configs, msg.Path)
	}

	// turn all our separate configurations into one
	univConfig := univcfg.MergeConfigs(configs)
	// generate new snapshot from configuration and update the cache
	err := eProcessor.Process(univConfig)
	if err != nil {
		return err
	}
	return nil
}
