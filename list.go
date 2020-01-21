package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	xdspb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes"
	"github.com/urfave/cli/v2"
)

func listClusters(c *cli.Context) error {
	cl, err := New(c, flgServer, flgNode)
	if err != nil {
		return err
	}
	defer cl.Stop()

	dr := xdspb.DiscoveryRequest{Node: cl.node}
	if c.String("c") != "" {
		dr.ResourceNames = []string{c.String("c")}
	}
	cds := xdspb.NewClusterDiscoveryServiceClient(cl.cc)
	resp, err := cds.FetchClusters(c.Context, &dr)
	if err != nil {
		return nil
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
	if len(clusters) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if flgHeader {
		fmt.Fprintln(w, "CLUSTER\tTYPE\t")
	}
	for _, u := range clusters {
		fmt.Fprintf(w, "%s\t%s\t\n", u.GetName(), u.GetType())
	}

	return nil
}

func listEndpoints(c *cli.Context) error {
	cl, err := New(c, flgServer, flgNode)
	if err != nil {
		return err
	}
	defer cl.Stop()

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if flgHeader {
		fmt.Fprintln(w, "CLUSTER\tENDPOINT\tSTATUS\tWEIGHT\t")
	}
	for _, e := range endpoints {
		endpoints := []string{}
		statuses := []string{}
		weights := []string{}
		for _, ep := range e.Endpoints {
			for _, lb := range ep.GetLbEndpoints() {
				port := strconv.Itoa(int(lb.GetEndpoint().GetAddress().GetSocketAddress().GetPortValue()))
				endpoints = append(endpoints, net.JoinHostPort(lb.GetEndpoint().GetAddress().GetSocketAddress().GetAddress(), port))
				statuses = append(statuses, corepb.HealthStatus_name[int32(lb.GetHealthStatus())])
				weight := strconv.Itoa(int(lb.GetLoadBalancingWeight().GetValue()))
				weights = append(weights, weight)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", e.GetClusterName(), strings.Join(endpoints, ","), strings.Join(statuses, ","), strings.Join(weights, ","))
	}

	return nil
}
