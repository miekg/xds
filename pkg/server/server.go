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

// Package server provides an implementation of a streaming xDS server.
package server

import (
	"context"

	"github.com/miekg/xds/pkg/cache"
)

//	healthpb.HealthDiscoveryServiceServer

// NewServer creates handlers from a config watcher and callbacks.
func NewServer(ctx context.Context, config *cache.Cluster) Server {
	return &server{cache: config, ctx: ctx}
}

type server struct {
	cache *cache.Cluster

	ctx context.Context

	// streamCount for counting bi-di streams
	streamCount int64
}
