package parser

import (
	"testing"

	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"
	usercfg "github.com/fmgornick/dynamic-envoy/utils/config/user"
	"github.com/stretchr/testify/assert"
)

var backends []usercfg.Backend = []usercfg.Backend{
	{
		Availability: []string{"internal"},
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{
				{
					Address: "internal.address1",
					Port:    1111,
				},
				{
					Address: "internal.address2:2222",
				},
			},
		},
	},
	{
		Availability: []string{"external"},
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{
				{
					Address: "external.address1",
					Port:    3333,
				},
				{
					Address: "external.address2",
				},
			},
		},
	},
	{
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{
				{
					Address: "http://both.address1",
				},
				{
					Address: "https://both.address2",
				},
			},
		},
	},
}

var bagWithoutId usercfg.Bag = usercfg.Bag{
	Availability: []string{"internal", "external"},
	Backends:     backends,
	Id:           "",
}
var internalBagWithoutId usercfg.Bag = usercfg.Bag{
	Availability: []string{"internal"},
	Backends:     backends,
	Id:           "",
}
var bagWithId usercfg.Bag = usercfg.Bag{
	Availability: []string{"internal", "external"},
	Backends:     backends,
	Id:           "bag",
}
var internalBagWithId usercfg.Bag = usercfg.Bag{
	Availability: []string{"internal"},
	Backends:     backends,
	Id:           "internal-bag",
}

var lconfig univcfg.ListenerInfo = univcfg.ListenerInfo{
	InternalAddress: "internal.address",
	ExternalAddress: "external.address",
	InternalPort:    1111,
	ExternalPort:    2222,
}
var parser BagParser = BagParser{
	Bags:         []usercfg.Bag{bagWithoutId, bagWithId},
	Config:       *univcfg.NewConfig(),
	ListenerInfo: lconfig,
}

func TestParse(t *testing.T) {
	var bags []usercfg.Bag = []usercfg.Bag{bagWithoutId, bagWithId}
	config, err := Parse(bags, lconfig)

	assert.NoError(t, err, "Parse should not produce an error")

	assert.Equal(t, "internal.address", config.Listeners["internal"].Address, "listener address should match")
	assert.Equal(t, "external.address", config.Listeners["external"].Address, "listener address should match")
	assert.Equal(t, uint(1111), config.Listeners["internal"].Port, "listener port should match")
	assert.Equal(t, uint(2222), config.Listeners["external"].Port, "listener port should match")

	assert.Equal(t, uint8(1), config.Clusters["in"].Availability, "in cluster should be internal")
	assert.Equal(t, uint8(2), config.Clusters["ex"].Availability, "ex cluster should be external")
}

func TestAddRoutes(t *testing.T) {
	p := BagParser{
		Bags: []usercfg.Bag{{
			Availability: []string{"internal", "external"},
			Backends: []usercfg.Backend{
				{
					Availability: []string{"internal"},
					Match: usercfg.Match{
						Path: usercfg.Path{
							Pattern: "/bag/path/internal/route",
							Type:    "starts_with",
						},
					},
					Server: usercfg.Server{
						Endpoints: []usercfg.Endpoint{{
							Address: "internal.endpoint.address",
							Port:    1111,
						}},
					},
				},
				{
					Availability: []string{"external"},
					Match: usercfg.Match{
						Path: usercfg.Path{
							Pattern: "",
						},
					},
					Server: usercfg.Server{
						Endpoints: []usercfg.Endpoint{{
							Address: "external.endpoint.address",
							Port:    2222,
						}},
					},
				},
				{
					Match: usercfg.Match{
						Path: usercfg.Path{
							Pattern: "/bag/path/route",
						},
					},
					Server: usercfg.Server{
						Endpoints: []usercfg.Endpoint{{
							Address: "both.endpoint.address",
							Port:    3333,
						}},
					},
				},
			},
			Id: "bag-path",
		}},
		Config:       *univcfg.NewConfig(),
		ListenerInfo: lconfig,
	}
	p.AddListeners()
	err1 := p.AddRoutes()
	assert.Equal(t, nil, err1, "AddRoutes should not produce an error")

	config := p.Config

	assert.Equal(t, "starts_with", config.Routes["bag-path-internal-route-in"].Type, "route should check prefix")
	assert.Equal(t, "exact", config.Routes["bag-path-ex"].Type, "route should check whole path")
	assert.Equal(t, "exact", config.Routes["bag-path-route-ie"].Type, "route should check whole path")

	assert.Equal(t, "/bag/path/internal/route", config.Routes["bag-path-internal-route-in"].Path, "path should match")
	assert.Equal(t, "/bag/path", config.Routes["bag-path-ex"].Path, "path should match")
	assert.Equal(t, "/bag/path/route", config.Routes["bag-path-route-ie"].Path, "path should match")

	p.Bags[0].Backends = append(p.Bags[0].Backends, usercfg.Backend{
		Availability: []string{"internal"},
		Match: usercfg.Match{
			Path: usercfg.Path{
				Pattern: "/invalid/path",
				Type:    "exact",
			},
		},
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{{
				Address: "invalid.endpoint.address",
				Port:    6666,
			}},
		},
	})

	err2 := p.AddRoutes()
	assert.EqualError(t, err2, "path pattern must start with \"/bag/path\"", "if pattern doesn't have id in start, should return error")
}

func TestAddEndpoints(t *testing.T) {
	err := parser.AddEndpoints()
	assert.Equal(t, nil, err, "AddEndpoints should not produce an error")

	config := parser.Config

	assert.Equal(t, "internal.address1", config.Endpoints["in"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "internal.address2", config.Endpoints["in"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(1111), config.Endpoints["in"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(2222), config.Endpoints["in"][1].Port, "port should be same (or retrieved from scheme)")

	assert.Equal(t, "external.address1", config.Endpoints["ex"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "external.address2", config.Endpoints["ex"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(3333), config.Endpoints["ex"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(443), config.Endpoints["ex"][1].Port, "port should be same (or retrieved from scheme)")

	assert.Equal(t, "http://both.address1", config.Endpoints["ie"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "https://both.address2", config.Endpoints["ie"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(80), config.Endpoints["ie"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(443), config.Endpoints["ie"][1].Port, "port should be same (or retrieved from scheme)")

	assert.Equal(t, "internal.address1", config.Endpoints["bag-in"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "internal.address2", config.Endpoints["bag-in"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(1111), config.Endpoints["bag-in"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(2222), config.Endpoints["bag-in"][1].Port, "port should be same (or retrieved from scheme)")

	assert.Equal(t, "external.address1", config.Endpoints["bag-ex"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "external.address2", config.Endpoints["bag-ex"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(3333), config.Endpoints["bag-ex"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(443), config.Endpoints["bag-ex"][1].Port, "port should be same (or retrieved from scheme)")

	assert.Equal(t, "http://both.address1", config.Endpoints["bag-ie"][0].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, "https://both.address2", config.Endpoints["bag-ie"][1].Address, "address should be same (plus ports stripped)")
	assert.Equal(t, uint(80), config.Endpoints["bag-ie"][0].Port, "port should be same (or retrieved from scheme)")
	assert.Equal(t, uint(443), config.Endpoints["bag-ie"][1].Port, "port should be same (or retrieved from scheme)")

	badBackend1 := []usercfg.Backend{{
		Availability: []string{"internal"},
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{
				{
					Address: "htp://internal.address1:1234",
					Port:    1111,
				},
			},
		},
	}}
	badBackend2 := []usercfg.Backend{{
		Availability: []string{"internal"},
		Server: usercfg.Server{
			Endpoints: []usercfg.Endpoint{
				{
					Address: "http://invalid.address2:port",
					Port:    2222,
				},
			},
		},
	}}

	badBag1 := []usercfg.Bag{{
		Availability: []string{"internal", "external"},
		Backends:     badBackend1,
		Id:           "",
	}}
	badBag2 := []usercfg.Bag{{
		Availability: []string{"internal", "external"},
		Backends:     badBackend2,
		Id:           "",
	}}

	badParser1 := BagParser{
		Bags:         badBag1,
		Config:       *univcfg.NewConfig(),
		ListenerInfo: lconfig,
	}
	badParser2 := BagParser{
		Bags:         badBag2,
		Config:       *univcfg.NewConfig(),
		ListenerInfo: lconfig,
	}

	err1 := badParser1.AddEndpoints()
	err2 := badParser2.AddEndpoints()

	assert.EqualError(t, err1, "invalid schema")
	assert.EqualError(t, err2, "invalid port")
}

func TestGetClusterName(t *testing.T) {
	new_backends := append(backends,
		usercfg.Backend{
			Availability: []string{"invalid"},
			Server: usercfg.Server{
				Endpoints: []usercfg.Endpoint{
					{
						Address: "invalid.address1",
						Port:    1111,
					},
					{
						Address: "invalid.address2",
						Port:    2222,
					},
				},
			},
		},
	)

	res1, err1 := getClusterName(bagWithoutId, new_backends[0])
	res2, err2 := getClusterName(bagWithoutId, new_backends[1])
	res3, err3 := getClusterName(bagWithoutId, new_backends[2])
	res4, err4 := getClusterName(bagWithoutId, new_backends[3])
	res5, err5 := getClusterName(internalBagWithoutId, new_backends[0])
	res6, err6 := getClusterName(internalBagWithoutId, new_backends[1])
	res7, err7 := getClusterName(internalBagWithoutId, new_backends[2])
	res8, err8 := getClusterName(internalBagWithoutId, new_backends[3])
	res9, err9 := getClusterName(bagWithId, new_backends[0])
	res10, err10 := getClusterName(bagWithId, new_backends[1])
	res11, err11 := getClusterName(bagWithId, new_backends[2])
	res12, err12 := getClusterName(bagWithId, new_backends[3])
	res13, err13 := getClusterName(internalBagWithId, new_backends[0])
	res14, err14 := getClusterName(internalBagWithId, new_backends[1])
	res15, err15 := getClusterName(internalBagWithId, new_backends[2])
	res16, err16 := getClusterName(internalBagWithId, new_backends[3])

	assert.Equal(t, "in", res1, "should only be in/ex/ie if no bag id")
	assert.NoError(t, err1, "should not produce an error")
	assert.Equal(t, "ex", res2, "should only be in/ex/ie if no bag id")
	assert.NoError(t, err2, "should not produce an error")
	assert.Equal(t, "ie", res3, "should only be in/ex/ie if no bag id")
	assert.NoError(t, err3, "should not produce an error")
	assert.Equal(t, "", res4, "nothing returned because it should produce an error")
	assert.EqualError(t, err4, "invalid element in backend availability array", "should fail because array has invalid value")
	assert.Equal(t, "in", res5, "should only be in/ex/ie if no bag id")
	assert.NoError(t, err5, "should not produce an error")
	assert.Equal(t, "", res6, "nothing returned because it should produce an error")
	assert.EqualError(t, err6, "bag and backend have conflicting availabilities", "should fail because availabilities don't match")
	assert.Equal(t, "in", res7, "should only be in/ex/ie if no bag id")
	assert.NoError(t, err7, "should not produce an error")
	assert.Equal(t, "", res8, "nothing returned because it should produce an error")
	assert.EqualError(t, err8, "invalid element in backend availability array", "should fail because array has invalid value")
	assert.Equal(t, "bag-in", res9, "should have bag prefix + availability extension")
	assert.NoError(t, err9, "should not produce an error")
	assert.Equal(t, "bag-ex", res10, "should have bag prefix + availability extension")
	assert.NoError(t, err10, "should not produce an error")
	assert.Equal(t, "bag-ie", res11, "should have bag prefix + availability extension")
	assert.NoError(t, err11, "should not produce an error")
	assert.Equal(t, "", res12, "nothing returned because it should produce an error")
	assert.EqualError(t, err12, "invalid element in backend availability array", "should fail because array has invalid value")
	assert.Equal(t, "internal-bag-in", res13, "should have internal-bag prefix + availability extension")
	assert.NoError(t, err13, "should not produce an error")
	assert.Equal(t, "", res14, "nothing returned because it should produce an error")
	assert.EqualError(t, err14, "bag and backend have conflicting availabilities", "should fail because availabilities don't match")
	assert.Equal(t, "internal-bag-in", res15, "should have internal-bag prefix + availability extension")
	assert.NoError(t, err15, "should not produce an error")
	assert.Equal(t, "", res16, "nothing returned because it should produce an error")
	assert.EqualError(t, err16, "invalid element in backend availability array", "should fail because array has invalid value")
}
