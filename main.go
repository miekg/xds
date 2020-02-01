// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

// Package main contains the test driver for testing xDS manually.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/miekg/xds/pkg/cache"
	"github.com/miekg/xds/pkg/resource"
	"github.com/miekg/xds/pkg/server"
)

var (
	mode     = flag.String("xds", resource.Ads, "Management server type (ads, xds, rest)")
	clusters = flag.Int("clusters", 4, "Number of clusters")
	nodeID   = flag.String("nodeID", "test-id", "Node ID")
	addr     = flag.String("addr", ":18000", "Management server address")
)

// main returns code 1 if any of the batches failed to pass all requests
func main() {
	flag.Parse()
	ctx := context.Background()

	// create a cache
	signal := make(chan struct{})
	cb := &callbacks{signal: signal}
	config := cache.NewSnapshotCache(*mode == resource.Ads, cache.IDHash{})
	srv := server.NewServer(context.Background(), config, cb)

	// create a test snapshot
	snapshots := resource.TestSnapshot{Xds: *mode, NumClusters: *clusters}

	go RunManagementServer(ctx, srv, *addr) // start the xDS server

	log.Println("waiting for the first request...")
	select {
	case <-signal:
		break
	case <-time.After(1 * time.Minute):
		log.Println("timeout waiting for the first request")
		os.Exit(1)
	}
	log.Printf("initial snapshot %+v\n", snapshots)

	snapshot := snapshots.Generate()
	if err := snapshot.Consistent(); err != nil {
		log.Printf("snapshot inconsistency: %+v\n", snapshot)
	}

	err := config.SetSnapshot(*nodeID, snapshot)
	if err != nil {
		log.Printf("snapshot error %q for %+v\n", err, snapshot)
		os.Exit(1)
	}
	snapshots.Version = fmt.Sprintf("v%d", 0)
	log.Printf("update snapshot %v\n", snapshots.Version)

	for {
		cb.Report()
		time.Sleep(5 * time.Second)
	}
}

type callbacks struct {
	signal   chan struct{}
	fetches  int
	requests int
	mu       sync.Mutex
}

func (cb *callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.Printf("server callbacks fetches=%d requests=%d\n", cb.fetches, cb.requests)
}
func (cb *callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	log.Printf("stream %d open for %s\n", id, typ)
	return nil
}
func (cb *callbacks) OnStreamClosed(id int64) {
	log.Printf("stream %d closed\n", id)
}
func (cb *callbacks) OnStreamRequest(int64, *xdspb.DiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.requests++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnStreamResponse(int64, *xdspb.DiscoveryRequest, *xdspb.DiscoveryResponse) {}
func (cb *callbacks) OnFetchRequest(_ context.Context, req *xdspb.DiscoveryRequest) error {
	println("FETCH", req.String())
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fetches++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnFetchResponse(_ *xdspb.DiscoveryRequest, resp *xdspb.DiscoveryResponse) {
	println("FETCH RESPONSE", resp.String())
}
