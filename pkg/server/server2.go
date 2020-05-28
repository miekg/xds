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
	cdspb "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverypb2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xdspb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	edspb "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/miekg/xds/pkg/log"
	"github.com/miekg/xds/pkg/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server2 is a collection of handlers for streaming discovery (v2) requests.
type Server2 interface {
	discoverypb2.AggregatedDiscoveryServiceServer

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
		apiVersion  = 0                   // version for this node
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

			// apiVersion is used to contruct the right endpoint discovery type when we send updates.
			switch req.TypeUrl {
			case resource.EndpointType3:
				apiVersion = 3
			case resource.EndpointType:
				apiVersion = 2
			}

			// type URL is required for ADS but is implicit for xDS
			if defaultTypeURL == resource.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			req2 := DiscoveryRequestToV3(req)
			resp, err := s.s.cache.Fetch(req2)
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
			versionInfo[req.TypeUrl] = resp.GetVersionInfo()
			log.Infof("Updated %s for node with ID %q with version: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])

		case <-tick.C:
			if apiVersion == 0 {
				log.Warningf("No API version seen from node with ID %s, defaulting to 2", node.Id)
			}
			req := &xdspb.DiscoveryRequest{}

			// CDS

			req.VersionInfo = versionInfo[resource.ClusterType3]
			req.TypeUrl = resource.ClusterType3
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
			log.Infof("Updated %s for node with ID %q with version: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])

			// EDS

			// depending on the version we need to look at different strings
			req.VersionInfo = versionInfo[resource.EndpointType3]
			req.TypeUrl = resource.EndpointType3
			if apiVersion == 0 || apiVersion == 2 {
				req.VersionInfo = versionInfo[resource.EndpointType]
				req.TypeUrl = resource.EndpointType
			}

			resp, err = s.s.cache.Fetch(req)
			if err != nil {
				return err
			}
			if resp.VersionInfo == versionInfo[req.TypeUrl] {
				log.Debugf("Update %s for node with ID %q not needed version up to date: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
				continue
			}

			resp2 = DiscoveryResponseToV2(resp)
			if err := send(resp2); err != nil {
				return err
			}
			versionInfo[req.TypeUrl] = resp.GetVersionInfo()
			log.Infof("Updated %s for node with ID %q with version: %s", req.TypeUrl, node.Id, versionInfo[req.TypeUrl])
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

func (s *server2) StreamEndpoints(stream edspb.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.discoveryHandler(stream, resource.EndpointType)
}

func (s *server2) StreamClusters(stream cdspb.ClusterDiscoveryService_StreamClustersServer) error {
	return s.discoveryHandler(stream, resource.ClusterType)
}

// Fetch is the universal fetch method.
func (s *server2) Fetch(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	req3 := DiscoveryRequestToV3(req)
	resp, err := s.s.cache.Fetch(req3)
	resp2 := DiscoveryResponseToV2(resp)
	return resp2, err
}

func (s *server2) FetchClusters(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	println("v2 fetch")
	return s.Fetch(ctx, req)
}

func (s *server2) FetchEndpoints(ctx context.Context, req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	println("v2 fetch")
	return s.Fetch(ctx, req)
}

func (s *server2) DeltaAggregatedResources(_ discoverypb2.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaEndpoints(_ edspb.EndpointDiscoveryService_DeltaEndpointsServer) error {
	return errors.New("not implemented")
}

func (s *server2) DeltaClusters(_ cdspb.ClusterDiscoveryService_DeltaClustersServer) error {
	return errors.New("not implemented")
}
