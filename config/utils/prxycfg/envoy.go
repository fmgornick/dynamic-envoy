package prxycfg

import (
	"fmt"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/wrapperspb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

// global mappings for envoy proxy
var pathType = map[string]string{
	"exact":       "path",
	"starts_with": "prefix",
}
var clusterPolicy = map[string]int32{
	"round_robin":      0,
	"least_request":    1,
	"ring_hash":        2,
	"random":           3,
	"maglev":           5,
	"cluster_provided": 6,
	"lb_policy_config": 7,
}

type Envoy struct {
	Listeners map[string]*listener.Listener
	Clusters  map[string]*cluster.Cluster
	Routes    map[string]*route.Route
	Endpoints map[string]*endpoint.LbEndpoint
}

func NewEnvoyConfig() *Envoy {
	return &Envoy{
		Listeners: make(map[string]*listener.Listener),
		Clusters:  make(map[string]*cluster.Cluster),
		Routes:    make(map[string]*route.Route),
		Endpoints: make(map[string]*endpoint.LbEndpoint),
	}
}

func (e *Envoy) MakeListener(address string, name string, port uint) {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
						ApiConfigSource: &core.ApiConfigSource{
							TransportApiVersion:       resource.DefaultAPIVersion,
							ApiType:                   core.ApiConfigSource_GRPC,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*core.GrpcService{{
								TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
								},
							}},
						},
					},
				},
				RouteConfigName: name + "_route",
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}
	e.Listeners[name] = &listener.Listener{
		Name: name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  address,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(port),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func (e *Envoy) MakeCluster(name string, policy string) {
	e.Clusters[name] = &cluster.Cluster{
		Name:                 name,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		LbPolicy:             cluster.Cluster_LbPolicy(clusterPolicy[policy]),
		// LoadAssignment:       makeEndpoint(clusterName, UpstreamHost),
		// DnsLookupFamily:  cluster.Cluster_V4_ONLY,
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ResourceApiVersion: resource.DefaultAPIVersion,
				ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &core.ApiConfigSource{
						TransportApiVersion:       resource.DefaultAPIVersion,
						ApiType:                   core.ApiConfigSource_GRPC,
						SetNodeOnFirstMessageOnly: true,
						GrpcServices: []*core.GrpcService{{
							TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
							},
						}},
					},
				},
			},
		},
	}
}

func (e *Envoy) MakeRoute(clusterName string, pathPattern string, pathType string) {
	switch pathType {
	case "prefix":
		e.Routes[clusterName] = &route.Route{
			Name: clusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: pathPattern,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: clusterName,
					},
				},
			},
		}
	case "path":
		e.Routes[clusterName] = &route.Route{
			Name: clusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Path{
					Path: pathPattern,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: clusterName,
					},
				},
			},
		}
	default:
		panic(fmt.Errorf("invalid path type"))
	}
}

func (e *Envoy) MakeEndpoint(address string, clusterName string, name string, port uint, weight uint) {
	var edp *endpoint.LbEndpoint
	if weight == 0 {
		edp = &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  address,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: uint32(port),
								},
							},
						},
					},
				},
			},
		}
	} else {
		edp = &endpoint.LbEndpoint{
			LoadBalancingWeight: &wrapperspb.UInt32Value{
				Value: uint32(weight),
			},
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  address,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: uint32(port),
								},
							},
						},
					},
				},
			},
		}
		e.Endpoints[name] = edp
		// e.Clusters[clusterName].LoadAssignment.Endpoints[0].LbEndpoints =
		// 	append(e.Clusters[clusterName].LoadAssignment.Endpoints[0].LbEndpoints, edp)
	}
}
