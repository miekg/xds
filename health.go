package main

import (
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/urfave/cli/v2"
)

// health sets the health for all endpoints in the cluster.
func health(c *cli.Context) error {
	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	// region zone subzone
	args := c.Args().Slice()
	if len(args) != 3 {
		return ErrArg(args)
	}
	region, zone, subzone := args[0], args[1], args[2]
	println("R", region, "Z", zone, "S", subzone)
	return nil
}

func healthNameToValue(h string) int32 {
	v, ok := corepb.HealthStatus_value[strings.ToUpper(h)]
	if !ok {
		return -1
	}
	return v
}
