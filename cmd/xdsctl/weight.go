package main

import (
	"fmt"
	"strconv"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/urfave/cli/v2"
)

// weight sets the weight for an endpoints in the cluster.
func weight(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) != 3 {
		return ErrArg(args)
	}

	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}

	cluster := args[0]
	endpoint := args[1]
	w := args[2]
	weight, err := strconv.ParseInt(w, 10, 32)
	if err != nil {
		return err
	}

	dr := &xdspb.DiscoveryRequest{Node: cl.node, ResourceNames: []string{cluster}}
	eds := edspb.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, dr)
	if err != nil {
		return err
	}
	lsr := &loadpb2.LoadStatsRequest{Node: &corepb2.Node{Id: cluster}}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			continue
		}
		c, ok := any.Message.(*xdspb2.ClusterLoadAssignment)
		if !ok {
			continue
		}

		for i := range c.Endpoints {
			for j := range c.Endpoints[i].LbEndpoints {
				ep := c.Endpoints[i].LbEndpoints[j].HostIdentifier.(*edspb2.LbEndpoint_Endpoint).Endpoint
				sa, ok := ep.Address.Address.(*corepb2.Address_SocketAddress)
				if !ok {
					return fmt.Errorf("endpoint %q does not contain a SocketAddress", ep)
				}
				addr := coreAddressToAddr(sa)
				if endpoint == "" || addr == endpoint {
					eh = append(eh, &healthpb.EndpointHealth{
						HealthStatus: corepb.HealthStatus(healthNameToValue(health)),
						Endpoint:     EndpointToV3(ep),
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
