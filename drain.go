package main

import (
	"github.com/urfave/cli/v2"
)

func drain(c *cli.Context) error {
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

func drainCluster(c *cli.Context) error { return nil }

func undrain(c *cli.Context) error {
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

func undrainCluster(c *cli.Context) error { return nil }
