package cache

import (
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes/duration"
)

// SetLoad sets the load for clusters and or endpoints. Adjust weights here as well??
func (c *Cluster) SetLoad(req *loadpb2.LoadStatsRequest) (*loadpb2.LoadStatsResponse, error) {
	clusters := []string{}
	for _, clusterStats := range req.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) == 0 {
			continue
		}
		clusters = append(clusters, clusterStats.ClusterName)
		// set the metadata
		//for _, upstreamStats := range clusterStats.UpstreamLocalityStats {}
		/*
			{
				UpstreamEndpointStats: []*edspb2.UpstreamEndpointStats{
					{
						Address:             endpoints[0].Address,
						TotalIssuedRequests: uint64(load),
					},
				},
			},
		*/
	}
	return &loadpb2.LoadStatsResponse{
		Clusters:                  []string{}, // empty list, to say this is for all clusters? Need to check how envoy deals with this.
		LoadReportingInterval:     &duration.Duration{Seconds: 2},
		ReportEndpointGranularity: true,
	}, nil
}
