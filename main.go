package main

import (
	"context"
	"fmt"

	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	test "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"
	usercfg "github.com/fmgornick/dynamic-envoy/utils/config/user"
	parser "github.com/fmgornick/dynamic-envoy/utils/parser"
	processor "github.com/fmgornick/dynamic-envoy/utils/processor"
	watcher "github.com/fmgornick/dynamic-envoy/utils/watcher"
	xdsServer "github.com/fmgornick/dynamic-envoy/utils/xdsServer"
)

const (
	directory = "./databags/local"
	xdsPort   = 6969
)

var configs map[string]*univcfg.Config
var snapConfig cache.SnapshotCache

func main() {
	configs = make(map[string]*univcfg.Config)
	snapConfig = cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	err := ProcessDir(snapConfig, watcher.Message{
		Operation: watcher.Create,
		Path:      directory,
	})
	if err != nil {
		panic(err)
	}

	// watch for file changes in specified directory
	change := make(chan watcher.Message)
	go func() {
		watcher.Watch(directory, change)
	}()

	go func() {
		server := server.NewServer(context.Background(), snapConfig, &test.Callbacks{})
		xdsServer.RunServer(context.Background(), server, xdsPort)
	}()

	for {
		select {
		case msg := <-change:
			fmt.Printf("processing new change...\n")
			err := ProcessDir(snapConfig, msg)
			if err != nil {
				err = fmt.Errorf("error processing new config: %+v\n", err)
				panic(err)
			}
			fmt.Printf("changes added!!!\n\n")
		}
	}
}

// TODO: fix this so we can dynamically change the configuration
func ProcessFile(snapConfig cache.SnapshotCache, msg watcher.Message) error {
	switch msg.Operation {
	case watcher.Create:
		bags, err := usercfg.ParseFile(msg.Path)
		if err != nil {
			return err
		}
		config, err := parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	case watcher.Modify:
		delete(configs, msg.Path)
		bags, err := usercfg.ParseFile(msg.Path)
		if err != nil {
			return err
		}
		config, err := parser.Parse(bags)
		if err != nil {
			return err
		}
		configs[msg.Path] = config

	default:
		delete(configs, msg.Path)
	}

	univConfig := univcfg.MergeConfigs(configs)
	snapshot, err := processor.Process(snapConfig, univConfig)
	if err != nil {
		return err
	}
	println("no")
	if err = snapConfig.SetSnapshot(context.Background(), "envoy-instance", snapshot); err != nil {
		return fmt.Errorf("snapshot error: %+v\n\n%+v", snapshot, err)
	}
	println("yes")
	return nil
}

func ProcessDir(snapConfig cache.SnapshotCache, file watcher.Message) error {
	bags, err := usercfg.ParseDir(directory)
	if err != nil {
		return err
	}
	univConfig, err := parser.Parse(bags)
	if err != nil {
		return err
	}
	_, err = processor.Process(snapConfig, univConfig)
	if err != nil {
		return err
	}
	return nil
}
