package main

import (
	"fmt"
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
	if len(args) < 2 || len(args) > 3 {
		return ErrArg(args)
	}

	cluster := args[0]
	endpoint := ""
	if len(args) == 3 {
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
	done := false
	endpoints := []*endpointpb.ClusterLoadAssignment{}
	hr := &healthpb.HealthCheckRequestOrEndpointHealthResponse{}
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
				// check endpoint name is given.
				endpoint = endpoint
				c.Endpoints[i].LbEndpoints[j].HealthStatus = corepb.HealthStatus(healthNameToValue(health))
				done = true
			}
		}
		endpoints = append(endpoints, c)
	}
	if !done {
		return fmt.Errorf("no matching endpoints found")
	}
	/*
		type EndpointHealthResponse struct {
			EndpointsHealth      []*EndpointHealth `protobuf:"bytes,1,rep,name=endpoints_health,json=endpointsHealth,proto3" json:"endpoints_health,omitempty"`
			XXX_NoUnkeyedLiteral struct{}          `json:"-"`
			XXX_unrecognized     []byte            `json:"-"`
			XXX_sizecache        int32             `json:"-"`
		}

		type EndpointHealth struct {
			Endpoint             *v31.Endpoint   `protobuf:"bytes,1,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
			HealthStatus         v3.HealthStatus `protobuf:"varint,2,opt,name=health_status,json=healthStatus,proto3,enum=envoy.config.core.v3.HealthStatus" json:"health_status,omitempty"`
			XXX_NoUnkeyedLiteral struct{}        `json:"-"`
			XXX_unrecognized     []byte          `json:"-"`
			XXX_sizecache        int32           `json:"-"`
		}
	*/

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
