package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/miekg/xds/pkg/cache"
	"github.com/miekg/xds/pkg/log"
	"github.com/miekg/xds/pkg/server"
)

var (
	nodeID = flag.String("nodeID", "test-id", "Node ID")
	addr   = flag.String("addr", ":18000", "management server address")
	conf   = flag.String("conf", ".", "cluster configuration directory")
)

// main returns code 1 if any of the batches failed to pass all requests
func main() {
	flag.Parse()
	clusters, err := parseClusters(*conf)
	if err != nil {
		log.Fatal(err)
	}
	// create a cache
	config := cache.New()
	for _, cla := range clusters {
		config.Insert(cla)
	}
	log.Infof("Initialized cache with 'v1' of %d cluster parsed from directory: %q", len(clusters), *conf)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	srv := server.NewServer(ctx, config)
	go RunManagementServer(ctx, srv, *addr) // start the xDS server

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			cancel()
			os.Exit(1)
		}
	}
}
