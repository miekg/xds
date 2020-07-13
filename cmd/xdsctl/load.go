package main

import (
	"fmt"
	"strconv"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/urfave/cli/v2"
)

// load sets the load for an endpoints in the cluster.
func load(c *cli.Context) error {
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
	load, err := strconv.ParseInt(w, 10, 32)
	if err != nil {
		return err
	}
	if load < 1 {
		return fmt.Errorf("load must be positive integer")
	}

	dr := &xdspb.DiscoveryRequest{Node: cl.node, ResourceNames: []string{cluster}}
	eds := edspb.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, dr)
	if err != nil {
		return err
	}

	// Technically we can just send in the report and let the server worry about the existence of this endpoint...

	// We search for the one endpoint, later we might introduce wildcards or stuff, like ignore the port?
	endpoints := []*edspb2.Endpoint{}
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
				if addr == endpoint {
					endpoints = append(endpoints, ep)
				}
			}
		}
	}
	if len(endpoints) == 0 {
		return fmt.Errorf("no matching endpoints found")
	}
	// do errors 'n stuff as well in the loadreport?
	// Hack alert: not filing out the locality.
	clstat := &edspb2.ClusterStats{
		ClusterName: cluster,
		UpstreamLocalityStats: []*edspb2.UpstreamLocalityStats{
			{
				UpstreamEndpointStats: []*edspb2.UpstreamEndpointStats{
					{
						Address:             endpoints[0].Address,
						TotalIssuedRequests: uint64(load),
					},
				},
			},
		},
		LoadReportInterval: &duration.Duration{Seconds: 2},
	}
	lr := &loadpb2.LoadStatsRequest{Node: &corepb2.Node{Id: cl.node.Id}, ClusterStats: []*edspb2.ClusterStats{clstat}}
	lrs := loadpb2.NewLoadReportingServiceClient(cl.cc)
	stream, err := lrs.StreamLoadStats(c.Context)
	if err != nil {
		return err
	}
	if err := stream.Send(lr); err != nil {
		return nil
	}
	_, err = stream.Recv()
	return err
}
