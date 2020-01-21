package main

import (
	xdspb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/urfave/cli/v2"
)

func setEndpoints(c *cli.Context) error {
	cl, err := New(c, c.String("s"), c.String("n"))
	if err != nil {
		return err
	}
	defer cl.Stop()

	// what here?
	dr := xdspb.DiscoveryRequest{Node: cl.node}
	if c.String("c") != "" {
		dr.ResourceNames = []string{c.String("c")}
	}
	eds := xdspb.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, &dr)
	if err != nil {
		return nil
	}

	endpoints := []*xdspb.ClusterLoadAssignment{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			continue
		}
		if c, ok := any.Message.(*xdspb.ClusterLoadAssignment); !ok {
			continue
		} else {
			endpoints = append(endpoints, c)

		}
	}
	if len(endpoints) == 0 {
		return nil
	}

	return nil
}
