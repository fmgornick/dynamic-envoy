package processor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/fmgornick/dynamic-envoy/config/utils/prxycfg"
	"github.com/fmgornick/dynamic-envoy/config/utils/univcfg"
)

type EnvoyProcessor struct {
	Config      univcfg.Config
	EnvoyConfig prxycfg.Envoy
	Cache       cache.SnapshotCache
	version     uint
}

func Process(config *univcfg.Config) (*cache.SnapshotCache, error) {
	var e EnvoyProcessor
	e.Config = *config
	e.EnvoyConfig = *prxycfg.NewEnvoyConfig()
	e.Cache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	e.version = 0

	var err error
	var snapshot *cache.Snapshot
	if snapshot, err = e.GenerateSnapshot(); err != nil {
		return nil, fmt.Errorf("couldn't generate snapshot: %+v", err)
	}
	if err = snapshot.Consistent(); err != nil {
		return nil, fmt.Errorf("snapshot inconsistency: %+v\n\n%+v", snapshot, err)
	}
	if err = e.Cache.SetSnapshot(context.Background(), "envoy-instance", *snapshot); err != nil {
		return nil, fmt.Errorf("snapshot error: %+v\n\n%+v", snapshot, err)
	}

	return &e.Cache, nil
}

// TODO: implement these functions to create snapshot
func (e *EnvoyProcessor) MakeListeners() []types.Resource {
	return nil
}

func (e *EnvoyProcessor) MakeClusters() []types.Resource {
	return nil
}

func (e *EnvoyProcessor) MakeRoutes() []types.Resource {
	return nil
}

func (e *EnvoyProcessor) MakeEndpoints() []types.Resource {
	return nil
}

func (e *EnvoyProcessor) GenerateSnapshot() (*cache.Snapshot, error) {
	snap, err := cache.NewSnapshot(e.newVersion(),
		map[resource.Type][]types.Resource{
			resource.ListenerType: e.MakeListeners(),
			resource.ClusterType:  e.MakeClusters(),
			resource.EndpointType: e.MakeEndpoints(),
			resource.RouteType:    e.MakeRoutes(),
		},
	)
	if err != nil {
		return nil, err
	}

	return &snap, nil
}

func (e *EnvoyProcessor) newVersion() string {
	e.version++
	return strconv.Itoa(int(e.version))
}

// // Parse file into object
// func (p *Processor) ProcessDir(dirname string) error {
// 	ParseDir(dirname)
// 	resource, err := ParseDir(dirname)
// 	if err != nil {
// 		return fmt.Errorf("trouble parsing json files: %+v", err)
// 	}
// 	p.Resource = *resource

// 	// Create the snapshot that we'll serve to Envoy
// 	snapshot, err := p.GenerateSnapshot()
// 	if err != nil {
// 		return fmt.Errorf("trouble creating snapshot: %+v", err)
// 	}

// 	if err := snapshot.Consistent(); err != nil {
// 		return fmt.Errorf("snapshot not consistent: %+v", err)
// 	}

// 	// Add the snapshot to the cache
// 	if err := p.Cache.SetSnapshot(context.Background(), "envoy-instance", *snapshot); err != nil {
// 		return fmt.Errorf("trouble setting snapshot: %+v", err)
// 	}

// 	return nil
// }
