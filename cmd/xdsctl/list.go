package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	cdspb "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/urfave/cli/v2"
)

func list(c *cli.Context) error {
	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}

	if c.NArg() > 0 {
		return listEndpoints(c)
	}

	dr := &xdspb.DiscoveryRequest{Node: cl.node, ResourceNames: c.Args().Slice()}
	cds := cdspb.NewClusterDiscoveryServiceClient(cl.cc)
	resp, err := cds.FetchClusters(c.Context, dr)
	if err != nil {
		return err
	}

	clusters := []*clusterpb.Cluster{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			return err
		}
		if c, ok := any.Message.(*clusterpb.Cluster); !ok {
			continue
		} else {
			clusters = append(clusters, c)
		}
	}
	if len(clusters) == 0 {
		return fmt.Errorf("no clusters found")
	}

	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Name < clusters[j].Name })

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if c.Bool("H") {
		fmt.Fprintln(w, "CLUSTER\tTYPE\tMETADATA\t")
	}
	for _, u := range clusters {
		fmt.Fprintf(w, "%s\t%s\t%s\t\n", u.GetName(), u.GetType(), strings.Join(metadataToStringSlice(u.GetMetadata()), " "))
	}

	return nil
}

func listEndpoints(c *cli.Context) error {
	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}

	dr := &xdspb.DiscoveryRequest{Node: cl.node, ResourceNames: c.Args().Slice()}
	eds := edspb.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, dr)
	if err != nil {
		return err
	}

	endpoints := []*endpointpb.ClusterLoadAssignment{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			return err
		}
		if c, ok := any.Message.(*endpointpb.ClusterLoadAssignment); !ok {
			// log ???
			continue
		} else {
			endpoints = append(endpoints, c)

		}
	}
	if len(endpoints) == 0 {
		return fmt.Errorf("no endpoints found")
	}

	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].ClusterName < endpoints[j].ClusterName })

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if c.Bool("H") {
		fmt.Fprintln(w, "CLUSTER\tENDPOINT\t\tLOCALITY\tSTATUS\tWEIGHT\t")
	}
	// we'll grab the data per localilty and then graph that. Locality is made up with Region/Zone/Subzone
	data := [][5]string{} // indexed by localilty and then numerical (0: name, 1: endpoints, 2: locality, 3: status, 4: weight)
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
			loc := ep.GetLocality()
			data = append(data, [5]string{
				e.GetClusterName(),
				strings.Join(endpoints, ","),
				strings.Join([]string{loc.GetRegion(), loc.GetZone(), loc.GetSubZone()}, " "),
				strings.Join(statuses, ","),
				strings.Join(weights, ","),
			})
		}

	}
	for i := range data {
		d := data[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", d[0], d[1], d[2], d[3], d[4])

	}
	return nil
}

// metadataToStringSlice converts the corepb.Metadata's Fields to a string slice where
// each elements is FIELD:VALUE. VALUE's type must be structpb.Value_StringValue otherwise
// it will be skipped.
func metadataToStringSlice(m *corepb.Metadata) []string {
	if m == nil {
		return nil
	}
	fields := []string{}
	for _, v := range m.FilterMetadata {
		for k, v1 := range v.Fields {
			v2, ok := v1.Kind.(*structpb.Value_StringValue)
			if !ok {
				continue
			}
			fields = append(fields, k+":"+v2.StringValue)
		}
	}
	return fields
}
