// FIX: external proxy configuration getting written over
package processor

import (
	"context"
	"fmt"
	"strconv"

	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	print "github.com/fmgornick/dynamic-envoy/config/utils/print"
	prxycfg "github.com/fmgornick/dynamic-envoy/config/utils/prxycfg"
	univcfg "github.com/fmgornick/dynamic-envoy/config/utils/univcfg"
)

type EnvoyProcessor struct {
	Config  univcfg.Config
	Cache   cache.SnapshotCache
	version uint
}

func Process(config *univcfg.Config) (*cache.SnapshotCache, error) {
	var e EnvoyProcessor
	e.Config = *config
	e.Cache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	var err error
	var snapshot *cache.Snapshot
	if snapshot, err = e.GenerateSnapshot(); err != nil {
		return nil, fmt.Errorf("couldn't generate snapshot: %+v", err)
	}
	if err = snapshot.Consistent(); err != nil {
		return nil, fmt.Errorf("snapshot inconsistency: \n\n%+v", err)
	}
	if err = e.Cache.SetSnapshot(context.Background(), "envoy-instance", *snapshot); err != nil {
		return nil, fmt.Errorf("snapshot error: %+v\n\n%+v", snapshot, err)
	}

	print.PrettyPrint(snapshot)
	return &e.Cache, nil
}

func (e *EnvoyProcessor) MakeListeners() []types.Resource {
	var resources []types.Resource

	for _, l := range e.Config.Listeners {
		resources = append(resources, prxycfg.MakeListener(l.Address, l.Name, l.Port))
	}

	return resources
}

func (e *EnvoyProcessor) MakeClusters() []types.Resource {
	var resources []types.Resource

	for _, c := range e.Config.Clusters {
		resources = append(resources, prxycfg.MakeCluster(c.Name, c.Policy))
	}

	return resources
}

func (e *EnvoyProcessor) MakeRoutes() []types.Resource {
	var internalRoutes []*route.Route
	var externalRoutes []*route.Route

	var resources []types.Resource

	for _, routeName := range e.Config.Listeners["internal"].Routes {
		r := e.Config.Routes[routeName]
		internalRoutes = append(internalRoutes, prxycfg.MakeRoute(r.ClusterName, r.Path, r.Type))
	}
	for _, routeName := range e.Config.Listeners["external"].Routes {
		r := e.Config.Routes[routeName]
		externalRoutes = append(externalRoutes, prxycfg.MakeRoute(r.ClusterName, r.Path, r.Type))
	}
	resources = append(resources, &route.RouteConfiguration{
		Name: "internal-routes",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "internal-routes",
			Domains: []string{"*"},
			Routes:  internalRoutes,
		}},
	})
	resources = append(resources, &route.RouteConfiguration{
		Name: "external-routes",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "external-routes",
			Domains: []string{"*"},
			Routes:  externalRoutes,
		}},
	})

	return resources
}

func (e *EnvoyProcessor) MakeEndpoints() []types.Resource {
	var resources []types.Resource
	var endpoints []*endpoint.LbEndpoint

	for clusterName, edps := range e.Config.Endpoints {
		for _, e := range edps {
			endpoints = append(endpoints, prxycfg.MakeEndpoint(e.Address, e.Port, e.Weight))
		}
		resources = append(resources, &endpoint.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: endpoints,
			}},
		})
	}

	return resources
}

func (e *EnvoyProcessor) GenerateSnapshot() (*cache.Snapshot, error) {
	snap, err := cache.NewSnapshot(e.newVersion(),
		map[resource.Type][]types.Resource{
			resource.ListenerType: e.MakeListeners(),
			resource.ClusterType:  e.MakeClusters(),
			resource.RouteType:    e.MakeRoutes(),
			resource.EndpointType: e.MakeEndpoints(),
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
