package cache

import (
	loadpb "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes/duration"
)

// SetLoad sets the load for clusters and or endpoints. Adjust weights here as well??
func (c *Cluster) SetLoad(req *loadpb.LoadStatsRequest) (*loadpb.LoadStatsResponse, error) {
	return &loadpb.LoadStatsResponse{
		Clusters:                  []string{req.GetNode().GetId()},
		LoadReportingInterval:     &duration.Duration{Seconds: 10},
		ReportEndpointGranularity: true,
	}, nil
}
