package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/grpc"
)

var (
	flgServer = flag.String("s", "", "server address to connect to")
	flgNode   = flag.String("n", "test-id", "node ID to use")
	flgClear  = flag.Bool("k", false, "don't use TLS")
	flgHeader = flag.Bool("h", true, "print header in output")
)

const padding = 3

func main() {
	flag.Parse()

	mandatory := []string{"s"}
	seen := make(map[string]struct{})

	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() != "" {
			seen[f.Name] = struct{}{}
		}
	})
	for _, m := range mandatory {
		if _, ok := seen[m]; !ok {
			errorf(fmt.Errorf("mandatory flag %q not set", m))
		}
	}

	args := flag.Args()
	if len(args) == 0 {
		errorf(fmt.Errorf("need verb"))
	}

	opts := []grpc.DialOption{}
	if *flgClear {
		opts = append(opts, grpc.WithInsecure())
	}
	c, err := New(*flgServer, *flgNode, opts...)
	if err != nil {
		errorf(err)
	}
	defer c.Stop()

	// version, list cluster|endpoints implemented
	switch args[0] {
	case "version":
		err = version(c, args[1:])
	case "list":
		err = list(c, args[1:])
	default:
		err = fmt.Errorf("wtf?")
	}

	if err != nil {
		errorf(err)
	}
}

func errorf(err error) {
	fmt.Printf("%s\n", err)
	os.Exit(1)
}
