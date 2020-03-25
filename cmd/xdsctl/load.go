package main

import (
	"github.com/urfave/cli/v2"
)

// load sets the load for an endpoints in the cluster.
func load(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) < 2 || len(args) > 3 {
		return ErrArg(args)
	}

	cl, err := New(c)
	if err != nil {
		return err
	}
	defer cl.Stop()

	if cl.dry {
		return nil
	}
	return nil
}
