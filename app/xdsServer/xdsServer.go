package xdsServer

import (
	"context"
	"fmt"
	"net"

	grpc "google.golang.org/grpc"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

// register services
func registerServer(grpcServer *grpc.Server, server server.Server) {
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
}

// start xds server on given port
func RunServer(ctx context.Context, server server.Server, port uint) {
	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	registerServer(grpcServer, server)

	fmt.Printf("xds server listening on port %d...\n\n", port)
	if err = grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
