package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const padding = 3

func main() {
	app := &cli.App{
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "s", Usage: "server `ADDRESS` to connect to", Value: "127.0.0.1:443"},
			&cli.StringFlag{Name: "n", Usage: "node `ID` to use`", Value: "test-id"},
			&cli.BoolFlag{Name: "k", Usage: "disable TLS"},
			&cli.BoolFlag{Name: "H", Usage: "print header in ouput", Value: true},
		},
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list clusters or endpoints, no arguments will list clusters",
				Action:  listClusters,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "c", Usage: "list cluster `NAME`"},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "clusters",
						Aliases: []string{"cluster"},
						Usage:   "list clusters",
						Action:  listClusters,
					},
					{
						Name:    "endpoints",
						Aliases: []string{"endpoint"},
						Usage:   "list endpoints",
						Action:  listEndpoints,
					},
				},
			},
			{
				Name:  "set",
				Usage: "set endoint's status in cluster",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "c", Usage: "use cluster `NAME`"},
				},
				Subcommands: []*cli.Command{
					{
						Name:   "endpoints",
						Usage:  "list endpoints",
						Action: listEndpoints,
					},
				},
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
