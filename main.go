package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/miekg/xds/pkg/cache"
	"github.com/miekg/xds/pkg/log"
	"github.com/miekg/xds/pkg/server"
)

var (
	nodeID = flag.String("nodeID", "test-id", "Node ID")
	addr   = flag.String("addr", ":18000", "management server address")
	conf   = flag.String("conf", ".", "cluster configuration directory")
	debug  = flag.Bool("debug", false, "enable debug logging")
)

// main returns code 1 if any of the batches failed to pass all requests
func main() {
	flag.Parse()
	clusters, err := parseClusters(*conf)
	if err != nil {
		log.Fatal(err)
	}
	if *debug {
		log.D.Set()
	}
	os.Exit(1)
	// create a cache
	config := cache.New()
	for _, cl := range clusters {
		config.Insert(cl)
	}
	log.Infof("Initialized cache with 'v1' of %d cluster parsed from directory: %q", len(clusters), *conf)

	// Every 10s look through the config directory to see if there are new files to be loaded
	stop := make(chan bool)
	go rereadConfig(config, *conf, stop)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	srv := server.NewServer(ctx, config)
	go RunManagementServer(ctx, srv, *addr) // start the xDS server

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			close(stop)
			cancel()
			os.Exit(1)
		}
	}
}

func rereadConfig(config *cache.Cluster, path string, stop <-chan bool) {
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			clusters, err := parseClusters(path)
			if err != nil {
				log.Warningf("Error reparsing clusters: %s", err)
				continue
			}
			current := config.All()
			for _, c := range clusters {
				i := sort.Search(len(current), func(i int) bool { return c.Name <= current[i] })
				if i < len(current) && current[i] == c.Name {
					continue
				}
				// new cluster
				log.Infof("Found new cluster in %q, adding cluster %s", path, c.Name)
				config.Insert(c)
			}
		}
	}
}
