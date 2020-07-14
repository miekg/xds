package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"

	"github.com/golang/protobuf/ptypes"
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

	dr := &xdspb2.DiscoveryRequest{Node: cl.node}
	cds := xdspb2.NewClusterDiscoveryServiceClient(cl.cc)
	resp, err := cds.FetchClusters(c.Context, dr)
	if err != nil {
		return err
	}

	clusters := []*xdspb2.Cluster{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			return err
		}
		if c, ok := any.Message.(*xdspb2.Cluster); ok { // v2
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
		fmt.Fprintln(w, "CLUSTER\tVERSION\tHEALTHCHECKS\t")
	}
	for _, u := range clusters {
		hcs := u.GetHealthChecks()
		hcname := []string{}
		for _, hc := range hcs {
			x := fmt.Sprintf("%T", hc.HealthChecker)
			// get the prefix of the name of the type
			// HealthCheck_HttpHealthCheck_ --> Http
			// and supper case it.
			prefix := strings.Index(x, "HealthCheck_")
			if prefix == -1 || len(x) < 11 {
				continue
			}
			name := strings.ToUpper(x[prefix+12:]) // get the last bit
			name = name[:len(name)-12]             // remove HealthCheck_
			hcname = append(hcname, name)

		}
		fmt.Fprintf(w, "%s\t%s\t%s\t\n", u.GetName(), resp.GetVersionInfo(), strings.Join(hcname, Joiner))
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

	args := c.Args().Slice()
	cluster := ""
	switch len(args) {
	default:
		return ErrArg(args)
	case 1:
		cluster = args[0]
	}

	// We can't use resource names here, because the API then assumes we care about
	// these and it will have a watch for it; if we then ask again we don't get any replies if
	// there isn't any updates to the clusters. So keep ResourceNames empty and we filter
	// down below.
	dr := &xdspb2.DiscoveryRequest{Node: cl.node}
	eds := xdspb2.NewEndpointDiscoveryServiceClient(cl.cc)
	resp, err := eds.FetchEndpoints(c.Context, dr)
	if err != nil {
		return err
	}

	endpoints := []*xdspb2.ClusterLoadAssignment{}
	for _, r := range resp.GetResources() {
		var any ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(r, &any); err != nil {
			return err
		}
		if c, ok := any.Message.(*xdspb2.ClusterLoadAssignment); ok {
			if cluster != "" && cluster != c.ClusterName {
				continue
			}
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
		fmt.Fprintln(w, "CLUSTER\tVERSION\tENDPOINT\tLOCALITY\tHEALTH\tWEIGHT\tLOAD")
	}
	// we'll grab the data per localilty and then graph that. Locality is made up with Region/Zone/Subzone
	data := [][6]string{} // indexed by localilty and then numerical (0: name, 1: endpoints, 2: locality, 3: status, 4: weight, 5: load)
	totalWeight := 0.0
	// same for load
	for _, e := range endpoints {
		for _, ep := range e.Endpoints {
			for _, lb := range ep.GetLbEndpoints() {
				totalWeight += float64(lb.GetLoadBalancingWeight().GetValue())
			}
		}
	}
	for _, e := range endpoints {
		for _, ep := range e.Endpoints {
			endpoints := []string{}
			healths := []string{}
			weights := []string{}
			loads := []string{}
			for _, lb := range ep.GetLbEndpoints() {
				port := strconv.Itoa(int(lb.GetEndpoint().GetAddress().GetSocketAddress().GetPortValue()))
				endpoints = append(endpoints, net.JoinHostPort(lb.GetEndpoint().GetAddress().GetSocketAddress().GetAddress(), port))
				healths = append(healths, corepb2.HealthStatus_name[int32(lb.GetHealthStatus())])
				weight := strconv.Itoa(int(lb.GetLoadBalancingWeight().GetValue()))
				// add fraction of total weight send to this endpoint
				frac := float64(lb.GetLoadBalancingWeight().GetValue()) / totalWeight
				weight = fmt.Sprintf("%s  %0.2f", weight, frac) // format: <weight> <total fraction>

				weights = append(weights, weight)
				loads = append(loads, loadFromMetadata(lb))
			}
			locs := []string{}
			loc := ep.GetLocality()
			if x := loc.GetRegion(); x != "" {
				locs = append(locs, x)
			}
			if x := loc.GetZone(); x != "" {
				locs = append(locs, x)
			}
			if x := loc.GetSubZone(); x != "" {
				locs = append(locs, x)
			}
			where := strings.TrimSpace(strings.Join(locs, Joiner))

			data = append(data, [6]string{
				e.GetClusterName(),
				strings.Join(endpoints, Joiner),
				where,
				strings.Join(healths, Joiner),
				strings.Join(weights, Joiner),
				strings.Join(loads, Joiner),
			})
		}

	}
	for _, d := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", d[0], resp.GetVersionInfo(), d[1], d[2], d[3], d[4])

	}
	return nil
}

func loadFromMetadata(lb *edspb2.LbEndpoint) string {
	if lb.Metadata == nil {
		return "0"
	}
	s, ok := lb.Metadata.FilterMetadata["load"] // we store the load here
	if !ok {
		return "0"
	}
	if s.Fields == nil {
		return "0"
	}
	sv := s.Fields["LOAD"] // 'LOAD' again, because nested maps
	return fmt.Sprintf("%d", uint64(sv.GetNumberValue()))
}

const Joiner = ","
