package cache

import (
	edspb3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	loadpb3 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/miekg/xds/pkg/log"
)

// SetWeight sets the weight within cluster for endpoints.
func (c *Cluster) SetWeight(req *loadpb3.LoadStatsRequest) (*loadpb3.LoadStatsResponse, error) {
	clusters := []string{}
	for _, clusterStats := range req.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) == 0 {
			continue
		}
		clusters = append(clusters, clusterStats.ClusterName)

		cl, _ := c.Retrieve(clusterStats.ClusterName)
		if cl == nil {
			// already checked if called from 'load', but this keep it here as a safeguard.
			log.Debugf("Weight report for unknown cluster %s", clusterStats.ClusterName)
			continue
		}

		done := false
		endpoints := cl.GetLoadAssignment()
		for _, upstreamStats := range clusterStats.UpstreamLocalityStats {
			for _, endpointStats := range upstreamStats.UpstreamEndpointStats {
				weight := WeightFromMetadata(endpointStats)
				if weight == 0 {
					log.Warningf("Expected weight to be set, got 0")
					continue
				}
				for _, ep := range endpoints.Endpoints {
					for _, lb := range ep.GetLbEndpoints() {
						epa := lb.GetEndpoint().GetAddress().GetSocketAddress()
						if epa.String() == endpointStats.Address.GetSocketAddress().String() {
							lb.LoadBalancingWeight = &wrapperspb.UInt32Value{Value: weight}
							done = true
						}
					}
				}
			}
		}
		if done {
			// we've updated something, write it back to the cache.
			c.Insert(cl)
			continue
		}
		log.Debug("Weight change for unknown endpoints in cluster %s", clusterStats.ClusterName)
	}
	return &loadpb3.LoadStatsResponse{
		Clusters:                  clusters,
		LoadReportingInterval:     &duration.Duration{Seconds: 2},
		ReportEndpointGranularity: true,
	}, nil
}

// WeightFromMetadata returns the weight from the metadata in the load report. If there is none, 0 is returned.
func WeightFromMetadata(us *edspb3.UpstreamEndpointStats) uint32 {
	if us.Metadata == nil || us.Metadata.Fields == nil {
		return 0
	}
	w, ok := us.Metadata.Fields[WeightKind]
	if !ok {
		return 0
	}
	return uint32(w.GetKind().(*structpb.Value_NumberValue).NumberValue)
}

func SetWeightInMetadata(us *edspb3.UpstreamEndpointStats, weight uint32) {
	if us.Metadata == nil {
		us.Metadata = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	us.Metadata.Fields[WeightKind] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(weight)}}
}
