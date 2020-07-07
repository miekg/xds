package server

// copied from envoy/examples/load-reporting-service/server

import (
	"sync/atomic"

	loadpb "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/miekg/xds/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// This is how often Envoy will send the load report
const StatsFrequencyInSeconds = 2

type loadStream interface {
	grpc.ServerStream

	Send(*loadpb.LoadStatsResponse) error
	Recv() (*loadpb.LoadStatsRequest, error)
}

// loadProcess handles a bi-di load stream request.
func (s *server2) loadProcess(stream loadStream, reqCh <-chan *loadpb.LoadStatsRequest) error {
	send := func(resp *loadpb.LoadStatsResponse) error {
		return stream.Send(resp)
	}

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
			nodeID := req.GetNode().GetId()
			// After Load Report is enabled, log the Load Report stats received
			for _, clusterStats := range req.ClusterStats {
				if len(clusterStats.UpstreamLocalityStats) > 0 {
					log.Debug("Got stats from cluster `%s` node `%s` - %s", req.Node.Cluster, nodeID, clusterStats)
				}
			}

			resp, err := s.s.cache.SetLoad(req)
			if err != nil {
				return err
			}
			return send(resp)
		}
	}
}

func (s *server2) loadHandler(stream loadStream) error {
	reqCh := make(chan *loadpb.LoadStatsRequest)
	reqStop := int32(0)
	go func() {
		for {
			req, err := stream.Recv()
			log.Debug("loadHandler called")
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

	err := s.loadProcess(stream, reqCh)
	atomic.StoreInt32(&reqStop, 1)
	return err
}

func (s *server2) StreamLoadStats(stream loadpb.LoadReportingService_StreamLoadStatsServer) error {
	log.Debug("StreamLoadStats called")
	return nil
}

/*
func (s *server) HandleRequest(stream loadpb.LoadReportingService_StreamLoadStatsServer, request *loadpb.LoadStatsRequest) {
	nodeID := request.GetNode().GetId()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether any Node has already connected or not.
	// If not, add the NodeID to nodesConnected and enable Load Report with given frequency
	// If yes, log stats
	if _, exist := s.nodesConnected[nodeID]; !exist {
		// Add NodeID to the nodesConnected
		log.Printf("Adding new new node to cache `%s`", nodeID)
		s.nodesConnected[nodeID] = true

		// Initialize Load Reporting
		err := stream.Send(&loadpb.LoadStatsResponse{
			Clusters:                  []string{"local_service"},
			LoadReportingInterval:     &duration.Duration{Seconds: StatsFrequencyInSeconds},
			ReportEndpointGranularity: true,
		})
		if err != nil {
			log.Panicf("Unable to send response to node %s due to err: %s", nodeID, err)
		}
		return
	}

	// After Load Report is enabled, log the Load Report stats received
	for _, clusterStats := range request.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) > 0 {
			log.Printf("Got stats from cluster `%s` node `%s` - %s", request.Node.Cluster, request.Node.Id, clusterStats)
		}
	}
}
*/
