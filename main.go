package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
var change chan watcher.Message          // used to keep track of changes to specified directory
var eProcessor *processor.EnvoyProcessor // used to send new configuration to envoy

func init() {
	// initialize environment variables, these can be set by user when running program via setting the flags
	flag.StringVar(&directory, "directory", "./databags/local", "path to folder containing databag files")
	flag.StringVar(&nodeId, "nodeId", "envoy-instance", "node id of envoy instance")
	flag.UintVar(&xdsPort, "port", 6969, "port number our xds management server is running on")

	// initialize our config map, directory watcher, and processor
	configs = make(map[string]*univcfg.Config)
	change = make(chan watcher.Message)
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
	// if file was deleted, immediately remove from configuration
	if msg.Operation == watcher.Move || msg.Operation == watcher.Delete {
		if configs[msg.Path] != nil {
			delete(configs, msg.Path)
		} else {
			for key := range configs {
				if strings.HasPrefix(key, msg.Path) {
					delete(configs, key)
				}
			}
		}
		// turn all our separate configurations into one
		univConfig := univcfg.MergeConfigs(configs)
		// generate new snapshot from configuration and update the cache
		err := eProcessor.Process(univConfig)
		if err != nil {
			return err
		}
		return nil
	} else {
		// check if file is a directory
		fileInfo, err := os.Stat(msg.Path)
		if err != nil {
			return fmt.Errorf("path check error: %+v", err)
		}

		// if it's a directory, then we want to call our operations on all the subdirectories and files
		// if it's a file, then we want to call ProcessFile, to actually update the config
		if fileInfo.IsDir() {
			return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Println(err)
					return nil
				}

				// don't want to recursively call ourself, otherwise it's an infinite loop
				println(path)
				if path == directory {
					return nil
				}

				// process change for all the subdirectories
				// if it's a file, then we want to call ProcessFile, to actually update the config
				if info.IsDir() {
					return ProcessChange(watcher.Message{
						Operation: msg.Operation,
						Path:      path,
					})
				} else {
					return ProcessFile(watcher.Message{
						Operation: msg.Operation,
						Path:      path,
					})
				}
			})
		} else {
			return ProcessFile(msg)
		}
	}
}

// called by ProcessChange, updates config of newly created/modified files
func ProcessFile(msg watcher.Message) error {
	var err error
	var bags []usercfg.Bag
	var config *univcfg.Config

	// what to do depending on operation...
	// created new file:   add it's configuration to our existing one
	// file changed:       delete existing configuration of file, then re-add it
	// file moved/deleted: delete existing configuration of file
	switch msg.Operation {
	case watcher.Create:
		// check if changed file is a directory or not
		bags, err = usercfg.ParseFile(msg.Path)
		if err != nil {
			return err
		}
		config, err = parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	case watcher.Modify:
		delete(configs, msg.Path)

		bags, err = usercfg.ParseFile(msg.Path)
		if err != nil {
			return err
		}
		config, err = parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	default:
		// delete and move should have been covered by ProcessChange
		return fmt.Errorf("invalid operation type")
	}

	// turn all our separate configurations into one
	univConfig := univcfg.MergeConfigs(configs)
	// generate new snapshot from configuration and update the cache
	err = eProcessor.Process(univConfig)
	if err != nil {
		return err
	}
	return nil
}
