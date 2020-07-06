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
	"fmt"
	"sync/atomic"

	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	"github.com/miekg/xds/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type healthStream interface {
	grpc.ServerStream

	Send(*healthpb.HealthCheckSpecifier) error
	Recv() (*healthpb.HealthCheckRequestOrEndpointHealthResponse, error)
}

// healthProcess handles a bi-di stream request.
func (s *server) healthProcess(stream healthStream, reqCh <-chan *healthpb.HealthCheckRequestOrEndpointHealthResponse) error {
	send := func(resp *healthpb.HealthCheckSpecifier) error {
		return stream.Send(resp)
	}

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
			hreq, ok := req.RequestType.(*healthpb.HealthCheckRequestOrEndpointHealthResponse_EndpointHealthResponse)
			if !ok {
				return status.Errorf(codes.Unavailable, "can only handle health check responses")
			}

			resp, err := s.cache.SetHealth(hreq.EndpointHealthResponse)
			if err != nil {
				return err
			}
			return send(resp)
		}
	}
}

// healthHandler converts a blocking read call to channels and initiates stream processing
func (s *server) healthHandler(stream healthStream) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *healthpb.HealthCheckRequestOrEndpointHealthResponse)
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

	err := s.healthProcess(stream, reqCh)
	atomic.StoreInt32(&reqStop, 1)
	return err
}

func (s *server) StreamHealthCheck(stream healthpb.HealthDiscoveryService_StreamHealthCheckServer) error {
	log.Debug("StreamHealthCheck called")
	return nil
}

func (s *server) FetchHealthCheck(ctx context.Context, req *healthpb.HealthCheckRequestOrEndpointHealthResponse) (*healthpb.HealthCheckSpecifier, error) {
	switch x := req.RequestType.(type) {
	case *healthpb.HealthCheckRequestOrEndpointHealthResponse_EndpointHealthResponse:
		return s.cache.SetHealth(x.EndpointHealthResponse)
	case *healthpb.HealthCheckRequestOrEndpointHealthResponse_HealthCheckRequest:
		return nil, fmt.Errorf("not implemented")
	}
	return nil, fmt.Errorf("not handled")
}
