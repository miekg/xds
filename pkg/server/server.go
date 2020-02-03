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
	"errors"
	"strconv"
	"sync/atomic"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cdspb "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	"github.com/miekg/xds/pkg/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is a collection of handlers for streaming discovery requests.
type Server interface {
	edspb.EndpointDiscoveryServiceServer
	cdspb.ClusterDiscoveryServiceServer
	discoverypb.AggregatedDiscoveryServiceServer
	healthpb.HealthDiscoveryServiceServer

	// Fetch is the universal fetch method for discovery requests
	Fetch(context.Context, *xdspb.DiscoveryRequest) (*xdspb.DiscoveryResponse, error)
}

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

type discoveryStream interface {
	grpc.ServerStream

	Send(*xdspb.DiscoveryResponse) error
	Recv() (*xdspb.DiscoveryRequest, error)
}

// discoveryProcess handles a bi-di stream request.
func (s *server) discoveryProcess(stream discoveryStream, reqCh <-chan *xdspb.DiscoveryRequest, defaultTypeURL string) error {
	// unique nonce generator for req-resp pairs per xDS stream; the server
	// ignores stale nonces. nonce is only modified within send() function.
	var streamNonce int64

	send := func(resp *discoverypb.DiscoveryResponse) error {
		streamNonce += 1
		resp.Nonce = strconv.FormatInt(streamNonce, 10)
		return stream.Send(resp)
	}

	// node may only be set on the first discovery request
	var node = &corepb.Node{}

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case req, more := <-reqCh:
			if !more { // input stream ended or errored out
				return nil
			}
			if req == nil {
				return status.Errorf(codes.Unavailable, "empty request")
			}

			// node field in discovery request is delta-compressed
			if req.Node != nil {
				node = req.Node
			} else {
				req.Node = node
			}

			// TODO, nonce checking
			// nonce := req.GetResponseNonce()

			// type URL is required for ADS but is implicit for xDS
			if defaultTypeURL == cache.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			resp, err := s.cache.Fetch(req)
			if err != nil {
				return err
			}
			return send(resp)
		}
	}
}

// discoveryHandler converts a blocking read call to channels and initiates stream processing
func (s *server) discoveryHandler(stream discoveryStream, typeURL string) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *xdspb.DiscoveryRequest)
	reqStop := int32(0)
	go func() {
		for {
			req, err := stream.Recv()
			if atomic.LoadInt32(&reqStop) != 0 {
				return
			}
			if err != nil {
				close(reqCh)
				return
			}
			reqCh <- req
		}
	}()

	err := s.discoveryProcess(stream, reqCh, typeURL)
	atomic.StoreInt32(&reqStop, 1)
	return err
}

func (s *server) StreamAggregatedResources(stream xdspb.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// health checks
	return s.discoveryHandler(stream, cache.AnyType)
}

func (s *server) StreamEndpoints(stream edspb.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.discoveryHandler(stream, cache.EndpointType)
}

func (s *server) StreamClusters(stream cdspb.ClusterDiscoveryService_StreamClustersServer) error {
	return s.discoveryHandler(stream, cache.ClusterType)
}

// Fetch is the universal fetch method.
func (s *server) Fetch(ctx context.Context, req *xdspb.DiscoveryRequest) (*xdspb.DiscoveryResponse, error) {
	// This could be extended further for health checks, but health checks write, so this may all be too
	// different.
	resp, err := s.cache.Fetch(req)
	return resp, err
}

func (s *server) FetchClusters(ctx context.Context, req *xdspb.DiscoveryRequest) (*xdspb.DiscoveryResponse, error) {
	req.TypeUrl = cache.ClusterType
	return s.Fetch(ctx, req)
}

func (s *server) FetchEndpoints(ctx context.Context, req *xdspb.DiscoveryRequest) (*xdspb.DiscoveryResponse, error) {
	req.TypeUrl = cache.EndpointType
	return s.Fetch(ctx, req)
}

func (s *server) DeltaAggregatedResources(_ xdspb.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaEndpoints(_ edspb.EndpointDiscoveryService_DeltaEndpointsServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaClusters(_ cdspb.ClusterDiscoveryService_DeltaClustersServer) error {
	return errors.New("not implemented")
}
