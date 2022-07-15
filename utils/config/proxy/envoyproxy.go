package prxycfg

import (
	"fmt"
	"time"

	anypb "google.golang.org/protobuf/types/known/anypb"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	wpb "google.golang.org/protobuf/types/known/wrapperspb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
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

// create listener envoyproxy configuration
func MakeHTTPSListener(address string, name string, port uint) *listener.Listener {
	routerpb, _ := anypb.New(&router.Router{})
	manager, _ := anypb.New(&hcm.HttpConnectionManager{
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
				// link internal listener to internal route configuration
				RouteConfigName: name + "-routes",
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: routerpb,
			},
		}},
	})
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
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: manager,
				},
			}},
			TransportSocket: transportSocket("downstream"),
		}},
	}
}

func MakeHTTPListener(address string, name string, port uint) *listener.Listener {
	routerpb, _ := anypb.New(&router.Router{})
	manager, _ := anypb.New(&hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				VirtualHosts: []*route.VirtualHost{{
					Name:    name + "-http-route",
					Domains: []string{"*"},
					Routes: []*route.Route{{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_Redirect{
							Redirect: &route.RedirectAction{
								SchemeRewriteSpecifier: &route.RedirectAction_HttpsRedirect{
									HttpsRedirect: true,
								},
								PathRewriteSpecifier: &route.RedirectAction_PathRedirect{
									PathRedirect: "/",
								},
							},
						},
					}},
				}},
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: routerpb,
			},
		}},
	})
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
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: manager,
				},
			}},
		}},
	}
}

// create cluster envoyproxy configuration
func MakeCluster(name string, policy string, useTLS bool) *cluster.Cluster {
	cluster := &cluster.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(5 * time.Second),
		// strict DNS is the only one that does multiple endpoints + ips or domains
		// logical DNS only does 1 enpoint
		// eds config only does IPs
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_LbPolicy(clusterPolicy[policy]),
	}
	if useTLS {
		cluster.TransportSocket = transportSocket("upstream")
	}
	return cluster
}

// create route envoyproxy configuration
func MakeRoute(clusterName string, pathPattern string, pathType string) *route.Route {
	// if we only care about the start of the path then we use the prefix match
	// if we care about the whole path then we use the path match
	action := &route.Route_Route{
		Route: &route.RouteAction{
			ClusterSpecifier: &route.RouteAction_Cluster{
				Cluster: clusterName,
			},
			PrefixRewrite: "/",
			HostRewriteSpecifier: &route.RouteAction_AutoHostRewrite{
				AutoHostRewrite: wpb.Bool(true),
			},
		},
	}
	switch pathType {
	case "starts_with":
		return &route.Route{
			Name: clusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: pathPattern,
				},
			},
			Action: action,
		}
	case "exact":
		return &route.Route{
			Name: clusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Path{
					Path: pathPattern,
				},
			},
			Action: action,
		}
	default:
		panic(fmt.Errorf("invalid path type"))
	}
}

// create endpoint envoyproxy configuration
func MakeEndpoint(address string, port uint, weight uint) *endpoint.LbEndpoint {
	// give the endpoints an assigned weight only if weight is specified
	// in the user configuration
	hid := &endpoint.LbEndpoint_Endpoint{
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
	}
	if weight == 0 {
		return &endpoint.LbEndpoint{
			HostIdentifier: hid,
		}
	} else {
		return &endpoint.LbEndpoint{
			LoadBalancingWeight: &wpb.UInt32Value{
				Value: uint32(weight),
			},
			HostIdentifier: hid,
		}
	}
}

func transportSocket(context string) *core.TransportSocket {
	commonTls := &tls.CommonTlsContext{
		TlsCertificates: []*tls.TlsCertificate{{
			CertificateChain: &core.DataSource{
				Specifier: &core.DataSource_Filename{
					Filename: "/etc/ssl/certs/cert.pem",
				},
			},
			PrivateKey: &core.DataSource{
				Specifier: &core.DataSource_Filename{
					Filename: "/etc/ssl/certs/key.pem",
				},
			},
		}},
		ValidationContextType: &tls.CommonTlsContext_ValidationContext{
			ValidationContext: &tls.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/etc/ssl/cert.pem",
					},
				},
			},
		},
	}

	var ctx *anypb.Any
	if context == "upstream" {
		ctx, _ = anypb.New(&tls.UpstreamTlsContext{CommonTlsContext: commonTls})
	} else {
		ctx, _ = anypb.New(&tls.DownstreamTlsContext{CommonTlsContext: commonTls})
	}

	return &core.TransportSocket{
		Name: wellknown.TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: ctx,
		},
	}
}
