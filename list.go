package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func list(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("empty verb")
	}
	switch args[0] {
	case "clusters":
		return listCluster(c, args[1:])
	case "endpoints":
		return listEndpoint(c, args[1:])
	}

	return fmt.Errorf("unknown verb: %q", args[0])
}

func listCluster(c *Client, args []string) error {
	stream, err := c.discovery(cdsURL, "", "", []string{})
	if err != nil {
		return err
	}
	clusters, err := c.receiveClusters(stream)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if *flgHeader {
		fmt.Fprintln(w, "CLUSTER\tTYPE\t")
	}
	for _, c := range clusters {
		fmt.Fprintf(w, "%s\t%s\t\n", c.GetName(), c.GetType())
	}

	return nil
}

func listEndpoint(c *Client, args []string) error {
	stream, err := c.discovery(edsURL, "", "", []string{})
	if err != nil {
		return err
	}
	endpoints, err := c.receiveEndpoints(stream)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	defer w.Flush()
	if *flgHeader {
		fmt.Fprintln(w, "CLUSTER\tENDPOINTS\tSTATUSES\tWEIGHTS\t")
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
