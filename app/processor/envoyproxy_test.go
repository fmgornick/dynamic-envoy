package processor

import (
	// "regexp"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	// endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	univcfg "github.com/fmgornick/dynamic-proxy/app/config/universal"
	watcher "github.com/fmgornick/dynamic-proxy/app/watcher"
)

var listenerInfo univcfg.ListenerInfo = univcfg.ListenerInfo{
	InternalAddress:    "internal.address",
	ExternalAddress:    "external.address",
	InternalPort:       uint(1111),
	ExternalPort:       uint(2222),
	InternalCommonName: "localhost",
	ExternalCommonName: "localhost",
}

func TestProcess(t *testing.T) {
	e := NewProcessor("node", false, listenerInfo)
	err := e.Process(watcher.Message{
		Operation: watcher.Create,
		Path:      "test_folder",
	})
	assert.Equal(t, nil, err, "function call should not produce error")

	config1 := e.Configs["test_folder/both.json"]
	config2 := e.Configs["test_folder/sub/internal.json"]
	config3 := e.Configs["test_folder/sub/external.json"]
	assert.NotEqual(t, nil, config1, "should have created path config")
	assert.NotEqual(t, nil, config2, "should have created path config")
	assert.NotEqual(t, nil, config3, "should have created path config")

	assert.Equal(t, "/", config2.Routes["in"].Path, "incorrect path for internal route")
	assert.Equal(t, "/", config3.Routes["ex"].Path, "incorrect path for internal route")

	assert.Equal(t, "internal.route", config2.Endpoints["in"][0].Address, "incorrect address for internal enpoint")
	assert.Equal(t, "external.route", config3.Endpoints["ex"][0].Address, "incorrect address for internal enpoint")
	assert.Equal(t, "external.route", config3.Endpoints["ex"][1].Address, "incorrect address for internal enpoint")
	assert.Equal(t, uint(4444), config2.Endpoints["in"][0].Port, "incorrect address for internal enpoint")
	assert.Equal(t, uint(5555), config3.Endpoints["ex"][0].Port, "incorrect address for internal enpoint")
	assert.Equal(t, uint(6666), config3.Endpoints["ex"][1].Port, "incorrect address for internal enpoint")
}

func TestProcessFile(t *testing.T) {
	e := NewProcessor("node", true, listenerInfo)
	err1 := e.processFile(watcher.Message{
		Operation: watcher.Create,
		Path:      "test_folder/both.json",
	})
	assert.Equal(t, nil, err1, "function call should not produce error")
	err2 := e.processFile(watcher.Message{
		Operation: watcher.Delete,
		Path:      "test_folder/both.json",
	})
	assert.EqualError(t, err2, "operation can only be modify or create", "should output invalid op error")

	config := e.Configs["test_folder/both.json"]
	assert.Equal(t, "internal.address", config.Listeners["internal"].Address, "incorrect address")
	assert.Equal(t, "external.address", config.Listeners["external"].Address, "incorrect address")
	assert.Equal(t, uint(1111), config.Listeners["internal"].Port, "incorrect port")
	assert.Equal(t, uint(2222), config.Listeners["external"].Port, "incorrect port")

	assert.Equal(t, uint8(1), config.Clusters["fletcher-3-in"].Availability, "incorrect availability for fletcher-3-in")
	assert.Equal(t, uint8(3), config.Clusters["fletcher-4-ie"].Availability, "incorrect availability for fletcher-4-ie")
	assert.Equal(t, "round_robin", config.Clusters["fletcher-3-in"].Policy, "incorrect lb policy for fletcher-3-in")
	assert.Equal(t, "round_robin", config.Clusters["fletcher-4-ie"].Policy, "incorrect lb policy for fletcher-4-ie")

	assert.Equal(t, "/fletcher/3", config.Routes["fletcher-3-in"].Path, "incorrect path for fletcher-3-in")
	assert.Equal(t, "/fletcher/4", config.Routes["fletcher-4-ie"].Path, "incorrect path for fletcher-4-ie")
	assert.Equal(t, "starts_with", config.Routes["fletcher-3-in"].Type, "incorrect path type for fletcher-3-in")
	assert.Equal(t, "exact", config.Routes["fletcher-4-ie"].Type, "incorrect path type for fletcher-4-ie")

	assert.Equal(t, "127.0.0.1", config.Endpoints["fletcher-3-in"][0].Address, "incorrect address for internal enpoint")
	assert.Equal(t, "127.0.0.1", config.Endpoints["fletcher-4-ie"][0].Address, "incorrect address for external enpoint")
	assert.Equal(t, uint(3333), config.Endpoints["fletcher-3-in"][0].Port, "incorrect port for internal enpoint")
	assert.Equal(t, uint(4444), config.Endpoints["fletcher-4-ie"][0].Port, "incorrect port for external enpoint")
}

func TestMakeRoutes(t *testing.T) {
	config := univcfg.NewConfig()
	config.AddRoute("cluster1-in", "/cluster1/path", "exact")
	config.AddRoute("cluster1-ie", "/cluster1/path", "exact")
	config.AddRoute("cluster2-ex", "/cluster2/path", "starts_with")
	config.AddListener("internal.address", "internal", 1111, "localhost")
	config.AddListener("external.address", "external", 2222, "localhost")
	config.Listeners["internal"].Routes = []string{"cluster1-in"}
	config.Listeners["external"].Routes = []string{"cluster1-ie", "cluster2-ex"}

	resources := makeRoutes(config)

	internalRoutes := resources[0].(*route.RouteConfiguration)
	externalRoutes := resources[1].(*route.RouteConfiguration)

	assert.Equal(t, "internal-routes", internalRoutes.Name, "should have name \"internal-routes\"")
	assert.Equal(t, "external-routes", externalRoutes.Name, "should have name \"external-routes\"")

	assert.IsType(t, &route.RouteMatch_Path{}, internalRoutes.VirtualHosts[0].Routes[0].Match.PathSpecifier,
		"should be downcasted to a path match")
	assert.IsType(t, &route.RouteMatch_Path{}, externalRoutes.VirtualHosts[0].Routes[0].Match.PathSpecifier,
		"should be downcasted to a path match")
	assert.IsType(t, &route.RouteMatch_Prefix{}, externalRoutes.VirtualHosts[0].Routes[1].Match.PathSpecifier,
		"should be downcasted to a prefix match")
	assert.Equal(t, "/cluster1/path",
		internalRoutes.VirtualHosts[0].Routes[0].Match.PathSpecifier.(*route.RouteMatch_Path).Path,
		"should match the path")
	assert.Equal(t, "/cluster1/path",
		externalRoutes.VirtualHosts[0].Routes[0].Match.PathSpecifier.(*route.RouteMatch_Path).Path,
		"should match the path")
	assert.Equal(t, "/cluster2/path",
		externalRoutes.VirtualHosts[0].Routes[1].Match.PathSpecifier.(*route.RouteMatch_Prefix).Prefix,
		"should match the path")
	assert.Equal(t, 1, len(internalRoutes.VirtualHosts[0].Routes), "should only contain 1 internal route")
}

func TestMakeEndpoints(t *testing.T) {
	config := univcfg.NewConfig()
	config.AddCluster("cluster1-in", "round_robin")
	config.AddCluster("cluster2-in", "round_robin")
	config.AddEndpoint("address1", "cluster1-in", 1111, "", 1)
	config.AddEndpoint("address2", "cluster2-in", 2222, "", 2)
	config.AddEndpoint("address3", "cluster1-in", 3333, "", 4)

	resources := makeClusters(config)
	loadAssignment1 := resources[0].(*clusterv3.Cluster).LoadAssignment
	loadAssignment2 := resources[1].(*clusterv3.Cluster).LoadAssignment

	if loadAssignment1.ClusterName != "cluster1-in" {
		loadAssignment1, loadAssignment2 = loadAssignment2, loadAssignment1
	}

	s := regexp.MustCompile(`\s+`)
	assert.Equal(t, 2, len(loadAssignment1.Endpoints[0].LbEndpoints), "should only have 2 lb enpoints")
	assert.Equal(t, "cluster1-in", loadAssignment1.ClusterName, "cluster name should match")
	assert.Equal(t, "address:\"address1\" port_value:1111",
		s.ReplaceAllString(loadAssignment1.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().String(), " "),
		"should have matching address and port")
	assert.Equal(t, "address:\"address3\" port_value:3333",
		s.ReplaceAllString(loadAssignment1.Endpoints[0].LbEndpoints[1].GetEndpoint().Address.GetSocketAddress().String(), " "),
		"should have matching address and port")

	assert.Equal(t, 1, len(loadAssignment2.Endpoints[0].LbEndpoints), "should only have 1 lb enpoint")
	assert.Equal(t, "cluster2-in", loadAssignment2.ClusterName, "cluster name should match")
	assert.Equal(t, "address:\"address2\" port_value:2222",
		s.ReplaceAllString(loadAssignment2.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().String(), " "),
		"should have matching address and port")
}
