package server

// copied from envoy/examples/load-reporting-service/server

import (
	"sync/atomic"

	loadpb "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	return s.loadHandler(stream)
}
