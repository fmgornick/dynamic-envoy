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

	// print "github.com/fmgornick/dynamic-envoy/config/utils/print"
	prxycfg "github.com/fmgornick/dynamic-envoy/utils/config/proxy"
	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"
)

type EnvoyProcessor struct {
	Config  univcfg.Config      // universal config (input)
	Cache   cache.SnapshotCache // snapshot config (output for envoyproxy)
	version uint                // keeps track of version number for our envoyproxy config
}

// take in the universal config and output the cache for envoyproxy
func Process(config *univcfg.Config) (*cache.SnapshotCache, error) {
	var e EnvoyProcessor
	e.Config = *config
	e.Cache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	var err error
	var snapshot *cache.Snapshot
	// this is the function that calls all the others to create the instance
	if snapshot, err = e.GenerateSnapshot(); err != nil {
		return nil, fmt.Errorf("couldn't generate snapshot: %+v", err)
	}
	// make sure our cache is consistent with itself
	if err = snapshot.Consistent(); err != nil {
		return nil, fmt.Errorf("snapshot inconsistency: \n\n%+v", err)
	}
	// set our cache
	if err = e.Cache.SetSnapshot(context.Background(), "envoy-instance", snapshot); err != nil {
		return nil, fmt.Errorf("snapshot error: %+v\n\n%+v", snapshot, err)
	}

	// return cache to the caller
	return &e.Cache, nil
}

// create resources array to hold all our listener configurations
func (e *EnvoyProcessor) makeListeners() []types.Resource {
	var resources []types.Resource

	for _, l := range e.Config.Listeners {
		resources = append(resources, prxycfg.MakeListener(l.Address, l.Name, l.Port))
	}

	return resources
}

// create resources array to hold all our cluster configurations
func (e *EnvoyProcessor) makeClusters() []types.Resource {
	var resources []types.Resource

	for _, c := range e.Config.Clusters {
		resources = append(resources, prxycfg.MakeCluster(c.Name, c.Policy))
	}

	return resources
}

// create resources array to hold all our route configurations
func (e *EnvoyProcessor) makeRoutes() []types.Resource {
	// keep track of internal and external routes
	var internalRoutes []*route.Route
	var externalRoutes []*route.Route

	var resources []types.Resource

	// iterate through internal routes listed in internal listener
	// add each route to our internal route array
	for _, routeName := range e.Config.Listeners["internal"].Routes {
		r := e.Config.Routes[routeName]
		internalRoutes = append(internalRoutes, prxycfg.MakeRoute(r.ClusterName, r.Path, r.Type))
	}
	// iterate through internal routes listed in external listener
	// add each route to our external route array
	for _, routeName := range e.Config.Listeners["external"].Routes {
		r := e.Config.Routes[routeName]
		externalRoutes = append(externalRoutes, prxycfg.MakeRoute(r.ClusterName, r.Path, r.Type))
	}
	// add internal route configuration to resources array for internal routes
	resources = append(resources, &route.RouteConfiguration{
		Name: "internal-routes",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "internal-routes",
			Domains: []string{"*"},
			Routes:  internalRoutes,
		}},
	})
	// add internal route configuration to resources array for external routes
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

// create resources array to hold all our endpoint configurations
func (e *EnvoyProcessor) makeEndpoints() []types.Resource {
	var resources []types.Resource
	var endpoints []*endpoint.LbEndpoint

	for clusterName, edps := range e.Config.Endpoints {
		// create endpoint array of all the endpoints that a single cluster maps to
		for _, e := range edps {
			endpoints = append(endpoints, prxycfg.MakeEndpoint(e.Address, e.Port, e.Weight))
		}
		// add this new array of endpoints to our resources array
		resources = append(resources, &endpoint.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: endpoints,
			}},
		})
	}

	return resources
}

// create our snapshot by calling all our private methods
func (e *EnvoyProcessor) GenerateSnapshot() (*cache.Snapshot, error) {
	snap, err := cache.NewSnapshot(e.newVersion(),
		map[resource.Type][]types.Resource{
			resource.ListenerType: e.makeListeners(),
			resource.ClusterType:  e.makeClusters(),
			resource.RouteType:    e.makeRoutes(),
			resource.EndpointType: e.makeEndpoints(),
		},
	)
	if err != nil {
		return nil, err
	}

	return snap, nil
}

func (e *EnvoyProcessor) newVersion() string {
	e.version++
	return strconv.Itoa(int(e.version))
}
