package prxycfg

import (
	"fmt"
	"time"

	ptypes "github.com/golang/protobuf/ptypes"
	wpb "google.golang.org/protobuf/types/known/wrapperspb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

var clusterPolicy = map[string]int32{
	"round_robin":      0,
	"least_request":    1,
	"ring_hash":        2,
	"random":           3,
	"maglev":           5,
	"cluster_provided": 6,
	"lb_policy_config": 7,
}

func MakeListener(address string, name string, port uint) *listener.Listener {
	router := &router.Router{}
	routerpb, err := ptypes.MarshalAny(router)
	if err != nil {
		panic(err)
	}
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
				RouteConfigName: name + "-routes",
			},
		},
		// NOTE: possible cause of problem
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: routerpb,
			},
		}},
	}
	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}
	return &listener.Listener{
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
		// NOTE: possible cause of problem
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

func MakeCluster(name string, policy string) *cluster.Cluster {
	return &cluster.Cluster{
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

func MakeRoute(clusterName string, pathPattern string, pathType string) *route.Route {
	switch pathType {
	case "starts_with":
		return &route.Route{
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
	case "exact":
		return &route.Route{
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

func MakeEndpoint(address string, port uint, weight uint) *endpoint.LbEndpoint {
	if weight == 0 {
		return &endpoint.LbEndpoint{
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
		return &endpoint.LbEndpoint{
			LoadBalancingWeight: &wpb.UInt32Value{
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
	}
}
