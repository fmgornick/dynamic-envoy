package print

import (
	"encoding/json"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
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
