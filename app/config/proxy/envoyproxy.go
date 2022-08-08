package prxycfg

import (
	"fmt"
	"time"

	univcfg "github.com/fmgornick/dynamic-proxy/app/config/universal"

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
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
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

var httpsPorts = map[string]uint{
	"internal": 48877,
	"external": 48878,
}

// create listener envoyproxy configuration
func MakeHTTPSListener(l *univcfg.Listener, hasHttp bool) *listener.Listener {
	var port *core.SocketAddress_PortValue
	if hasHttp {
		port = &core.SocketAddress_PortValue{
			PortValue: uint32(httpsPorts[l.Name]),
		}
	} else {
		port = &core.SocketAddress_PortValue{
			PortValue: uint32(l.Port),
		}
	}

	routerpb, _ := anypb.New(&router.Router{})
	manager, _ := anypb.New(&hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "https",
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
				RouteConfigName: l.Name + "-routes",
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
		Name: "https-" + l.Name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol:      core.SocketAddress_TCP,
					Address:       l.Address,
					PortSpecifier: port,
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
			TransportSocket: transportSocket(l.CommonName),
		}},
	}
}

func MakeHTTPListener(l *univcfg.Listener) []*listener.Listener {
	routerpb, _ := anypb.New(&router.Router{})
	manager, _ := anypb.New(&hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				VirtualHosts: []*route.VirtualHost{{
					Name:    l.Name + "-http-route",
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
								PortRedirect: uint32(httpsPorts[l.Name]),
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
	http_listener := &listener.Listener{
		Name: "http-" + l.Name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  l.Address,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(l.Port),
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
	https_listener := MakeHTTPSListener(l, true)

	return []*listener.Listener{http_listener, https_listener}
}

// create cluster envoyproxy configuration
// TODO: find a way to get hostname for healthcheck
func MakeCluster(c *univcfg.Cluster, https bool) *cluster.Cluster {
	cluster := &cluster.Cluster{
		Name:           c.Name,
		ConnectTimeout: durationpb.New(5 * time.Second),
		// strict DNS is the only one that does multiple endpoints + ips or domains
		// logical DNS only does 1 enpoint
		// eds config only does IPs
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_LbPolicy(clusterPolicy[c.Policy]),
	}
	if https {
		cluster.TransportSocket = transportSocket()
	}
	if c.HealthCheck != nil {
		cluster.HealthChecks = []*core.HealthCheck{{
			Timeout:            &durationpb.Duration{Seconds: int64(5)},
			Interval:           &durationpb.Duration{Seconds: int64(c.HealthCheck.Interval)},
			UnhealthyThreshold: &wpb.UInt32Value{Value: uint32(c.HealthCheck.Unhealthy)},
			HealthyThreshold:   &wpb.UInt32Value{Value: uint32(c.HealthCheck.Healthy)},
			HealthChecker:      &core.HealthCheck_HttpHealthCheck_{},
		}}
		if c.HealthCheck.Type == "http" {
			if c.HealthCheck.Host == "" {
				cluster.HealthChecks[0].HealthChecker = &core.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
						Path: c.HealthCheck.Path,
					},
				}
			} else {
				cluster.HealthChecks[0].HealthChecker = &core.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
						Path: c.HealthCheck.Path,
						Host: c.HealthCheck.Host,
					},
				}
			}
		} else {
			cluster.HealthChecks[0].HealthChecker = &core.HealthCheck_TcpHealthCheck_{
				TcpHealthCheck: &core.HealthCheck_TcpHealthCheck{},
			}
		}

	}
	return cluster
}

// create route envoyproxy configuration
func MakeRoute(r *univcfg.Route) *route.Route {
	// if we only care about the start of the path then we use the prefix match
	// if we care about the whole path then we use the path match
	action := &route.Route_Route{
		Route: &route.RouteAction{
			ClusterSpecifier: &route.RouteAction_Cluster{
				Cluster: r.ClusterName,
			},
			HostRewriteSpecifier: &route.RouteAction_AutoHostRewrite{
				AutoHostRewrite: wpb.Bool(true),
			},
		},
	}
	switch r.Type {
	case "starts_with":
		return &route.Route{
			Name: r.ClusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: r.Path,
				},
			},
			Action: action,
		}
	case "exact":
		return &route.Route{
			Name: r.ClusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Path{
					Path: r.Path,
				},
			},
			Action: action,
		}
	case "regex":
		return &route.Route{
			Name: r.ClusterName,
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_SafeRegex{
					SafeRegex: &matcher.RegexMatcher{
						EngineType: &matcher.RegexMatcher_GoogleRe2{},
						Regex:      r.Path,
					},
				},
			},
			Action: action,
		}
	default:
		panic(fmt.Errorf("invalid path type in clustername: %s", r.ClusterName))
	}
}

// create endpoint envoyproxy configuration
func MakeEndpoint(e *univcfg.Endpoint) *endpoint.LbEndpoint {
	// give the endpoints an assigned weight only if weight is specified
	// in the user configuration
	hid := &endpoint.LbEndpoint_Endpoint{
		Endpoint: &endpoint.Endpoint{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  e.Address,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(e.Port),
						},
					},
				},
			},
		},
	}
	if e.Weight == 0 {
		return &endpoint.LbEndpoint{
			HostIdentifier: hid,
		}
	} else {
		return &endpoint.LbEndpoint{
			LoadBalancingWeight: &wpb.UInt32Value{
				Value: uint32(e.Weight),
			},
			HostIdentifier: hid,
		}
	}
}

func transportSocket(cName ...string) *core.TransportSocket {
	var ctx *anypb.Any
	if len(cName) == 0 {
		ctx, _ = anypb.New(&tls.UpstreamTlsContext{})
	} else {
		ctx, _ = anypb.New(&tls.DownstreamTlsContext{CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificates: []*tls.TlsCertificate{{
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "certs/" + cName[0] + ".crt",
					},
				},
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "certs/" + cName[0] + ".key",
					},
				},
			}},
		}})
	}

	return &core.TransportSocket{
		Name: wellknown.TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: ctx,
		},
	}
}
