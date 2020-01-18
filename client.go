package main

import (
	"context"
	"fmt"
	"os"

	xdspb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	adsgrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc"
)

const (
	cdsURL = "type.googleapis.com/envoy.api.v2.Cluster"
	edsURL = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
)

type adsStream adsgrpc.AggregatedDiscoveryService_StreamAggregatedResourcesClient

// Client talks to the grpc manager's endpoint.
type Client struct {
	cc     *grpc.ClientConn
	node   *corepb.Node
	ctx    context.Context
	cancel context.CancelFunc
}

// New returns a new client that's dialed to addr using node as the local identifier.
func New(addr, node string, opts ...grpc.DialOption) (*Client, error) {
	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	hostname, _ := os.Hostname()
	c := &Client{cc: cc, node: &corepb.Node{Id: node,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"HOSTNAME": {
					Kind: &structpb.Value_StringValue{StringValue: hostname},
				},
			},
		},
		BuildVersion: Version,
	},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return c, nil
}

func (c *Client) Stop() error { c.cancel(); return c.cc.Close() }

func (c *Client) discovery(typeURL, version, nonce string, clusters []string) (adsStream, error) {
	// setting up a stream for a cli is a but dumb, but this is the code I have and I'm just copying (and it works)
	cli := adsgrpc.NewAggregatedDiscoveryServiceClient(c.cc)
	stream, err := cli.StreamAggregatedResources(c.ctx)
	if err != nil {
		return stream, err
	}

	req := &xdspb.DiscoveryRequest{
		Node:          c.node,
		TypeUrl:       typeURL,
		ResourceNames: clusters,
		VersionInfo:   version,
		ResponseNonce: nonce,
	}
	if err := stream.Send(req); err != nil {
		return stream, err
	}
	return stream, nil
}

func (c *Client) receiveEndpoints(stream adsStream) ([]*xdspb.ClusterLoadAssignment, error) {
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	if resp.GetTypeUrl() != edsURL {
		return nil, fmt.Errorf("wrong response URL for endpoint discovery: %q", resp.GetTypeUrl())
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
	return endpoints, nil
}

func (c *Client) receiveClusters(stream adsStream) ([]*xdspb.Cluster, error) {
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	if resp.GetTypeUrl() != cdsURL {
		return nil, fmt.Errorf("wrong response URL for cluster discovery: %q", resp.GetTypeUrl())
	}

	clusters := []*xdspb.Cluster{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			continue
		}
		if c, ok := any.Message.(*xdspb.Cluster); !ok {
			continue
		} else {
			clusters = append(clusters, c)
		}
	}
	return clusters, nil
}
