package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/urfave/cli/v2"
)

func health(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) < 2 || len(args) > 3 {
		return ErrArg(args)
	}
	if len(args) == 2 {
		return healthStatus(c, args[1])
	}
	return healthStatus(c, args[2])
}

// health sets the health for all endpoints in the cluster.
func healthStatus(c *cli.Context, health string) error {
	if healthNameToValue(health) == -1 {
		return fmt.Errorf("unknown type of health: %s", health)
	}

	args := c.Args().Slice()
	if len(args) < 1 || len(args) > 3 {
		return ErrArg(args)
	}

	cluster := args[0]
	endpoint := ""
	if len(args) >= 2 {
		endpoint = args[1]
	}

	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}

	dr := &xdspb.DiscoveryRequest{Node: cl.node, ResourceNames: []string{cluster}}
	eds := edspb.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, dr)
	if err != nil {
		return err
	}
	// Get the endpoints for this cluster, then either set them all to health or just the
	// one that matches.
	eh := []*healthpb.EndpointHealth{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			continue
		}
		c, ok := any.Message.(*endpointpb.ClusterLoadAssignment)
		if !ok {
			continue
		}
		for i := range c.Endpoints {
			for j := range c.Endpoints[i].LbEndpoints {
				ep := c.Endpoints[i].LbEndpoints[j].HostIdentifier.(*endpointpb.LbEndpoint_Endpoint).Endpoint
				sa, ok := ep.Address.Address.(*corepb.Address_SocketAddress)
				if !ok {
					return fmt.Errorf("endpoint %q does not contain a SocketAddress", ep)
				}
				addr := coreAddressToAddr(sa)
				if endpoint == "" || addr == endpoint {
					eh = append(eh, &healthpb.EndpointHealth{
						HealthStatus: corepb.HealthStatus(healthNameToValue(health)),
						Endpoint:     ep,
					})
				}
			}
		}
	}
	if len(eh) == 0 {
		return fmt.Errorf("no matching endpoints found")
	}

	hr := &healthpb.HealthCheckRequestOrEndpointHealthResponse{
		RequestType: &healthpb.HealthCheckRequestOrEndpointHealthResponse_EndpointHealthResponse{
			EndpointHealthResponse: &healthpb.EndpointHealthResponse{
				EndpointsHealth: eh,
			},
		},
	}
	hds := healthpb.NewHealthDiscoveryServiceClient(cl.cc)
	_, err = hds.FetchHealthCheck(c.Context, hr)
	return err
}

func healthNameToValue(h string) int32 {
	v, ok := corepb.HealthStatus_value[strings.ToUpper(h)]
	if !ok {
		return -1
	}
	return v
}

func coreAddressToAddr(sa *corepb.Address_SocketAddress) string {
	addr := sa.SocketAddress.Address

	port, ok := sa.SocketAddress.PortSpecifier.(*corepb.SocketAddress_PortValue)
	if !ok {
		return addr
	}
	return net.JoinHostPort(addr, strconv.FormatUint(uint64(port.PortValue), 10))
}
