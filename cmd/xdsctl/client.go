package main

import (
	"os"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const (
	cdsURL = "type.googleapis.com/envoy.api.v3.Cluster"
	edsURL = "type.googleapis.com/envoy.api.v3.ClusterLoadAssignment"
)

// Client talks to the grpc manager's endpoint.
type Client struct {
	cc   *grpc.ClientConn
	node *corepb.Node
	dry  bool
}

// New returns a new client that's dialed to addr using node as the local identifier.
// if flgClear is set grpc.WithInsecure is added to opts.
func New(c *cli.Context, opts ...grpc.DialOption) (*Client, error) {
	hostname, _ := os.Hostname()
	node := &corepb.Node{Id: c.String("n"), Metadata: &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"HOSTNAME":     {Kind: &structpb.Value_StringValue{StringValue: hostname}},
			"BUILDVERSION": {Kind: &structpb.Value_StringValue{StringValue: c.String("v")}},
		},
	}}
	if c.Bool("N") { // dryrun
		return &Client{node: node, dry: true}, nil
	}
	if c.Bool("k") {
		opts = append(opts, grpc.WithInsecure())
	}

	cc, err := grpc.Dial(c.String("s"), opts...)
	if err != nil {
		return nil, err
	}
	return &Client{cc: cc, node: node}, nil
}

func (c *Client) Stop() error {
	if c.dry {
		return nil
	}
	return c.cc.Close()
}
