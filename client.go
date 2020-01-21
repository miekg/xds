package main

import (
	"os"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	adsgrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const (
	cdsURL = "type.googleapis.com/envoy.api.v2.Cluster"
	edsURL = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
)

type adsStream adsgrpc.AggregatedDiscoveryService_StreamAggregatedResourcesClient

// Client talks to the grpc manager's endpoint.
type Client struct {
	cc   *grpc.ClientConn
	node *corepb.Node
}

// New returns a new client that's dialed to addr using node as the local identifier.
// if flgClear is set grpc.WithInsecure is added to opts.
func New(c *cli.Context, addr, node string, opts ...grpc.DialOption) (*Client, error) {
	if c.Bool("k") {
		opts = append(opts, grpc.WithInsecure())
	}
	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	hostname, _ := os.Hostname()
	cl := &Client{cc: cc, node: &corepb.Node{Id: node,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"HOSTNAME": {
					Kind: &structpb.Value_StringValue{StringValue: hostname},
				},
			},
		},
		BuildVersion: c.String("v"),
	},
	}
	return cl, nil
}

func (c *Client) Stop() error { return c.cc.Close() }
