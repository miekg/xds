package main

import (
	"fmt"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
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
func healthStatus(c *cli.Context, status string) error {
	// cluster [endpoint] health
	args := c.Args().Slice()
	if len(args) < 2 || len(args) > 3 {
		return ErrArg(args)
	}
	// list cluster and then ....
	cluster := args[0]
	endpoint := ""
	health := ""
	if len(args) == 2 {
		health = args[1]
	}
	if len(args) == 3 {
		endpoint = args[1]
		health = args[2]
	}
	if healthNameToValue(health) == -1 {
		return fmt.Errorf("unknown type of health: %s", health)
	}

	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}
	hr := &healthpb.HealthCheckRequestOrEndpointHealthResponse{}
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