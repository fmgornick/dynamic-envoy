package print

import (
	"encoding/json"

	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	univcfg "github.com/fmgornick/dynamic-proxy/app/config/universal"
)

func PrintMap(m map[string][]types.Resource) {
	listener := m[resource.ListenerType]
	cluster := m[resource.ClusterType]
	route := m[resource.RouteType]

	println("LISTENER")
	println("--------")
	PrettyPrint(listener)
	println()
	println()
	println("CLUSTER")
	println("--------")
	PrettyPrint(cluster)
	println()
	println()
	println("ROUTE")
	println("--------")
	PrettyPrint(route)
}

func PrettyPrint(data interface{}) {
	d, _ := json.MarshalIndent(data, "", "  ")
	println(string(d))
}

func EnvoyPrint(configs map[string]*univcfg.Config) {
	cfg := univcfg.MergeConfigs(configs)
	PrettyPrint(cfg)
}
