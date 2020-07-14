package cache

import (
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/miekg/xds/pkg/log"
)

// SetLoad sets the load for clusters and or endpoints.
func (c *Cluster) SetLoad(req *loadpb2.LoadStatsRequest) (*loadpb2.LoadStatsResponse, error) {
	clusters := []string{}
	for _, clusterStats := range req.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) == 0 {
			continue
		}
		clusters = append(clusters, clusterStats.ClusterName)

		cl, _ := c.Retrieve(clusterStats.ClusterName)
		if cl == nil {
			// we don't know this cluster
			log.Debugf("Load report for unknown cluster %s", clusterStats.ClusterName)
			continue
		}
		done := false
		endpoints := cl.GetLoadAssignment()

		for _, upstreamStats := range clusterStats.UpstreamLocalityStats {
			for _, endpointStats := range upstreamStats.UpstreamEndpointStats {
				for _, ep := range endpoints.Endpoints {
					for _, lb := range ep.GetLbEndpoints() {
						epa := lb.GetEndpoint().GetAddress().GetSocketAddress()
						if epa.String() == endpointStats.Address.GetSocketAddress().String() {
							SetLoadInMetadata(lb, float64(endpointStats.TotalIssuedRequests))
							done = true
						}
					}
				}
			}
		}
		if done {
			// we've updated something, write it back to the cache.
			c.Insert(cl)
		}
	}
	return &loadpb2.LoadStatsResponse{
		Clusters:                  clusters,
		LoadReportingInterval:     &duration.Duration{Seconds: 2},
		ReportEndpointGranularity: true,
	}, nil
}

// LoadFromMetada returns the load as reported in the metadata of the endpoint.
func LoadFromMetadata(lb *edspb2.LbEndpoint) float64 {
	if lb.Metadata == nil {
		return 0
	}
	s, ok := lb.Metadata.FilterMetadata["load"] // we store the load here
	if !ok {
		return 0
	}
	if s.Fields == nil {
		return 0
	}
	sv := s.Fields["LOAD"] // 'LOAD' again, because nested maps
	return sv.GetNumberValue()
}

// SetLoadInMetadata adds load to the metadata of the LbEndpoint.
func SetLoadInMetadata(lb *edspb2.LbEndpoint, load float64) {
	if lb.Metadata == nil {
		lb.Metadata = new(corepb2.Metadata)
	}
	if lb.Metadata.FilterMetadata == nil {
		lb.Metadata.FilterMetadata = map[string]*structpb.Struct{}
	}
	s, ok := lb.Metadata.FilterMetadata["load"]
	if !ok {
		lb.Metadata.FilterMetadata["load"] = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		lb.Metadata.FilterMetadata["load"].Fields["LOAD"] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: load}}
		return
	}
	s.Fields["LOAD"].GetKind().(*structpb.Value_NumberValue).NumberValue += load
}
