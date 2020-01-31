package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const padding = 3

func main() {
	app := &cli.App{
		Version: "0.0.2",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "s", Usage: "server `ADDRESS` to connect to", Required: true},
			&cli.StringFlag{Name: "n", Usage: "node `ID` to use", Value: "test-id"},
			&cli.BoolFlag{Name: "k", Usage: "disable TLS"},
			&cli.BoolFlag{Name: "H", Usage: "print header in output", Value: true},
			&cli.BoolFlag{Name: "N", Usage: "dry run"},
			&cli.BoolFlag{Name: "d", Usage: "dump protocol buffers to standard output"},
		},
		// load and locale (currently not set)
		Commands: []*cli.Command{
			{
				Name: "ls",
				Description: "List lists clusters and endpoints of clusters. If no endpoint is given the entire cluster is listed.\n" +
					"   If not cluster is given, all clusters are shown.",
				Usage:     "list (all) clusters and endpoints",
				ArgsUsage: "[CLUSTERS [ENDPOINT]]",
				Action:    list,
			},
			{
				Name: "drain",
				Description: "Drain sets the endpoint's health to DRAINING. If no endpoint is given all endpoints for this cluster will be set.\n" +
					"   When clusters share endpoints they will get updated as well.",
				Category:  "health",
				Usage:     "set health status to DRAINING for endpoints or entire clusters",
				ArgsUsage: "CLUSTER [ENDPOINT]",
				Action: func(c *cli.Context) error {
					err := healthStatus(c, "DRAINING")
					return err
				},
			},
			{
				Name: "undrain",
				Description: "Undrain sets the endpoint's health to UNKNOWN. If no endpoint is given all endpoints for this cluster will be set.\n" +
					"   When clusters share endpoints they will get updated as well.",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "i", Usage: "set endpoint immediately to HEALTHY"},
				},
				Category:  "health",
				Usage:     "set health status to UNKNOWN for endpoints or entire clusters",
				ArgsUsage: "CLUSTER [ENDPOINT]",
				Action: func(c *cli.Context) error {
					status := "UNKNOWN"
					if c.Bool("i") {
						status = "HEALTHY"
					}
					err := healthStatus(c, status)
					return err
				},
			},
			{
				Name: "health",
				Description: "Health sets the health for endpoints in a cluster. If no endpoint is given all endpoints for this cluster will be set.\n" +
					"   The mandatory argument HEALTH_STATUS can be: 'UNKNOWN', 'HEALTHY', 'UNHEALTHY', 'DRAINING', 'TIMEOUT' or 'DEGRADED'.\n" +
					"   When clusters share endpoint they will get updated as well.",
				Category:  "health",
				ArgsUsage: "CLUSTER [ENDPOINT] HEALTH_STATUS",
				Usage:     "set health status for endpoints or entire clusters",
				Action:    health,
			},
			{
				Name:        "weight",
				Description: "Weight sets the endpoint's weight.",
				Usage:       "set endpoint's weight in a cluster",
				ArgsUsage:   "CLUSTER ENDPOINT",
				Action:      weight,
			},
			{
				Name:        "add",
				Description: "Add adds clusters and endpoints. A new endpoint will have its health set to UNKNOWN.",
				Category:    "admin",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Usage:       "add a cluster or add a cluster and endpoint",
				Action:      add,
			},
			{
				Name:        "rm",
				Description: "Remove removes clusters and endpoints. If no endpoint is given the entire cluster is removed.",
				Category:    "admin",
				Usage:       "remove  a cluster or remove a cluster and endpoint",
				ArgsUsage:   "CLUSTER [ENDPOINT]",
				Action:      remove,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		errorf(err)
	}
}

func errorf(err error) {
	fmt.Printf("%s\n", err)
	os.Exit(1)
}

var (
	ErrArg = func(s []string) error { return fmt.Errorf("argument parse error: %v", s) }
)
