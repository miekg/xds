package server

// this file implements the v2 version of the xds protocol

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discoverypb2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/miekg/xds/pkg/log"
	"github.com/miekg/xds/pkg/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server2 is a collection of handlers for streaming discovery (v2) requests.
type Server2 interface {
	discoverypb2.AggregatedDiscoveryServiceServer
	xdspb2.EndpointDiscoveryServiceServer
	xdspb2.ClusterDiscoveryServiceServer
	xdspb2.ListenerDiscoveryServiceServer
	xdspb2.RouteDiscoveryServiceServer
	loadpb2.LoadReportingServiceServer
	// healthpb.HealthDiscoveryServiceServer -- this is still the v3 bit

	// Server2 is only a wrapper around the actual server; it mostly translates protobufs to the
	// right format and then calls the right method of server.

	// Fetch is the universal fetch method for discovery requests
	Fetch(context.Context, *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error)
}

func NewServer2(s Server) Server2 {
	return &server2{s: s.(*server)}
}

type server2 struct {
	s *server
}

type discoveryStream2 interface {
	grpc.ServerStream

	Send(*xdspb2.DiscoveryResponse) error
	Recv() (*xdspb2.DiscoveryRequest, error)
}

// discoveryProcess handles a bi-di stream (v2) request.
func (s *server2) discoveryProcess(stream discoveryStream2, reqCh <-chan *xdspb2.DiscoveryRequest, defaultTypeURL string) error {
	// This function is copied from the server.go file. I think we can make things work in an even more transparant way
	// but for now we'll just copy and paste code around.
	var streamNonce int64

	send := func(resp *xdspb2.DiscoveryResponse) error {
		streamNonce += 1
		resp.Nonce = strconv.FormatInt(streamNonce, 10)
		return stream.Send(resp)
	}

	tick := time.NewTicker(10 * time.Second) // every 10s we send updates (if there are any to this client).
	defer tick.Stop()

	var (
		node        = &corepb2.Node{}
		versionInfo = map[string]string{} // API string -> version CDS/EDS
	)

	for {
		select {
		case <-s.s.ctx.Done():
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

			// type URL is required for ADS but is implicit for xDS
			if defaultTypeURL == resource.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			req3 := DiscoveryRequestToV3(req)
			resp, err := s.s.cache.Fetch(req3)
			if err != nil {
				return err
			}
			resp2 := DiscoveryResponseToV2(resp)
			if resp.VersionInfo == versionInfo[req.TypeUrl] {
				log.Debugf("Update %s for node with ID %q not needed version up to date: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
				continue
			}

			if err := send(resp2); err != nil {
				return err
			}
			log.Infof("Updated %s for node with ID %q with version: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
			versionInfo[req.TypeUrl] = resp.GetVersionInfo()
		case <-tick.C:
			req := &xdspb.DiscoveryRequest{}

			for _, tpy := range []string{resource.ClusterType, resource.EndpointType, resource.ListenerType, resource.RouteConfigType} {
				req.VersionInfo = versionInfo[tpy]
				req.TypeUrl = tpy
				resp, err := s.s.cache.Fetch(req)
				if err != nil {
					return err
				}
				if resp.VersionInfo == versionInfo[req.TypeUrl] {
					log.Debugf("Update %s for node with ID %q not needed version up to date: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
					continue
				}

				resp2 := DiscoveryResponseToV2(resp)
				if err := send(resp2); err != nil {
					return err
				}
				versionInfo[req.TypeUrl] = resp.GetVersionInfo()
				log.Infof("updated %s for node with ID %q with version: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
			}
		}
	}
}

// discoveryHandler converts a blocking read call to channels and initiates stream processing.
func (s *server2) discoveryHandler(stream discoveryStream2, typeURL string) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *xdspb2.DiscoveryRequest)
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

func (s *server2) StreamAggregatedResources(stream discoverypb2.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return s.discoveryHandler(stream, resource.AnyType)
}

func (s *server2) StreamEndpoints(stream xdspb2.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.discoveryHandler(stream, resource.EndpointType)
}

func (s *server2) StreamClusters(stream xdspb2.ClusterDiscoveryService_StreamClustersServer) error {
	return s.discoveryHandler(stream, resource.ClusterType)
}

func (s *server2) StreamListeners(stream xdspb2.ListenerDiscoveryService_StreamListenersServer) error {
	return s.discoveryHandler(stream, resource.ListenerType)
}

func (s *server2) StreamRoutes(stream xdspb2.RouteDiscoveryService_StreamRoutesServer) error {
	return s.discoveryHandler(stream, resource.RouteConfigType)
}

// Fetch is the universal fetch method.
func (s *server2) Fetch(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req3 := DiscoveryRequestToV3(req)
	resp, err := s.s.cache.Fetch(req3)
	resp2 := DiscoveryResponseToV2(resp)
	return resp2, err
}

func (s *server2) FetchClusters(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req.TypeUrl = resource.ClusterType
	return s.Fetch(ctx, req)
}

func (s *server2) FetchEndpoints(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req.TypeUrl = resource.EndpointType
	return s.Fetch(ctx, req)
}

func (s *server2) FetchListeners(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req.TypeUrl = resource.ListenerType
	return s.Fetch(ctx, req)
}

func (s *server2) FetchRoutes(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req.TypeUrl = resource.RouteConfigType
	return s.Fetch(ctx, req)
}

func (s *server2) DeltaAggregatedResources(_ discoverypb2.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaEndpoints(_ xdspb2.EndpointDiscoveryService_DeltaEndpointsServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaClusters(_ xdspb2.ClusterDiscoveryService_DeltaClustersServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaListeners(_ xdspb2.ListenerDiscoveryService_DeltaListenersServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaRoutes(_ xdspb2.RouteDiscoveryService_DeltaRoutesServer) error {
	return errors.New("not implemented")
}
