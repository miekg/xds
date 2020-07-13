package cache

import (
	loadpb "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/miekg/xds/pkg/log"
)

// SetLoad sets the load for clusters and or endpoints. Adjust weights here as well??
func (c *Cluster) SetLoad(req *loadpb.LoadStatsRequest) (*loadpb.LoadStatsResponse, error) {
	// we should check if we have the cluster, and then do something with the load.
	// depending on LBPolicy we do different things with it?? Or just adjust the weights.
	nodeID := req.GetNode().GetId()
	println(nodeID)
	for _, clusterStats := range req.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) > 0 {
			log.Debugf("Got stats from cluster `%s` node `%s` - %s", req.Node.Cluster, nodeID, clusterStats)
		}
	}
	return &loadpb.LoadStatsResponse{
		Clusters:                  []string{}, // empty list, to say this is for all clusters? Need to check how envoy deals with this.
		LoadReportingInterval:     &duration.Duration{Seconds: 2},
		ReportEndpointGranularity: true,
	}, nil
}
