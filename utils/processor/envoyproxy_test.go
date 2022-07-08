package processor

import (
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMakeEndpoints(t *testing.T) {
	assert.Equal(t, 123, 123, "these should be equal")

	config := univcfg.NewConfig()
	config.AddEndpoint("address1", "cluster1", 1111, "", 1)
	config.AddEndpoint("address2", "cluster2", 2222, "", 2)
	config.AddEndpoint("address3", "cluster1", 3333, "", 3)

	resources := makeEndpoints(config)

	cluster1 := resources[0].(*endpoint.ClusterLoadAssignment)
	cluster2 := resources[1].(*endpoint.ClusterLoadAssignment)

	assert.Equal(t, 2, len(cluster1.Endpoints[0].LbEndpoints), "should only have 2 lb enpoints")
	assert.Equal(t, "cluster1", cluster1.ClusterName, "cluster name should match")
	assert.Equal(t, "address:\"address1\"  port_value:1111",
		cluster1.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().String(),
		"should have matching address and port")
	assert.Equal(t, "address:\"address3\"  port_value:3333",
		cluster1.Endpoints[0].LbEndpoints[1].GetEndpoint().Address.GetSocketAddress().String(),
		"should have matching address and port")

	assert.Equal(t, 1, len(cluster2.Endpoints[0].LbEndpoints), "should only have 1 lb enpoint")
	assert.Equal(t, "cluster2", cluster2.ClusterName, "cluster name should match")
	assert.Equal(t, "address:\"address2\"  port_value:2222",
		cluster2.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().String(),
		"should have matching address and port")
}
