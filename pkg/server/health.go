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

	healthpb2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type healthStream interface {
	grpc.ServerStream

	Send(*healthpb2.HealthCheckSpecifier) error
	Recv() (*healthpb2.HealthCheckRequestOrEndpointHealthResponse, error)
}

// healthProcess handles a bi-di stream request.
func (s *server) healthProcess(stream healthStream, reqCh <-chan *healthpb2.HealthCheckRequestOrEndpointHealthResponse) error {
	send := func(resp *healthpb2.HealthCheckSpecifier) error {
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
			hreq, ok := req.RequestType.(*healthpb2.HealthCheckRequestOrEndpointHealthResponse_EndpointHealthResponse)
			if !ok {
				return status.Errorf(codes.Unavailable, "can only handle health check responses")
			}

			resp, err := s.cache.SetHealth(hreq.EndpointHealthResponse)
			if err != nil {
				return err
			}
			if err := send(resp); err != nil {
				return err
			}
		}
	}
}

// healthHandler converts a blocking read call to channels and initiates stream processing
func (s *server) healthHandler(stream healthStream) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *healthpb2.HealthCheckRequestOrEndpointHealthResponse)
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

func (s *server) StreamHealthCheck(stream healthpb2.HealthDiscoveryService_StreamHealthCheckServer) error {
	return nil
}

func (s *server) FetchHealthCheck(ctx context.Context, req *healthpb2.HealthCheckRequestOrEndpointHealthResponse) (*healthpb2.HealthCheckSpecifier, error) {
	switch x := req.RequestType.(type) {
	case *healthpb2.HealthCheckRequestOrEndpointHealthResponse_EndpointHealthResponse:
		return s.cache.SetHealth(x.EndpointHealthResponse)
	case *healthpb2.HealthCheckRequestOrEndpointHealthResponse_HealthCheckRequest:
		return nil, fmt.Errorf("not implemented")
	}
	return nil, fmt.Errorf("not handled")
}
