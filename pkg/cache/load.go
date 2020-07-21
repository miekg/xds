package cache

import (
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	loadpb3 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/miekg/xds/pkg/log"
)

// SetLoad sets the load for clusters and or endpoints.
func (c *Cluster) SetLoad(req *loadpb3.LoadStatsRequest) (*loadpb3.LoadStatsResponse, error) {
	clusters := []string{}
	for _, clusterStats := range req.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) == 0 {
			continue
		}
		clusters = append(clusters, clusterStats.ClusterName)

		cl, _ := c.Retrieve(clusterStats.ClusterName)
		if cl == nil {
			log.Debugf("Load report for unknown cluster %s", clusterStats.ClusterName)
			continue
		}

		// Is this our hack to set the weights via load reporting?
		for _, upstreamStats := range clusterStats.UpstreamLocalityStats {
			for _, endpointStats := range upstreamStats.UpstreamEndpointStats {
				weight := WeightFromMetadata(endpointStats)
				// if one of them has it we assume the entire things is about changing weights
				if weight > 0 {
					return c.SetWeight(req)
				}
			}
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
			continue
		}
		log.Debug("Load report for unknown endpoints in cluster %s", clusterStats.ClusterName)
	}
	return &loadpb3.LoadStatsResponse{
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
	s, ok := lb.Metadata.FilterMetadata[LoadKind] // we store the load here
	if !ok {
		return 0
	}
	if s.Fields == nil {
		return 0
	}
	sv := s.Fields[LoadKind] // 'load' again, because nested maps
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
	s, ok := lb.Metadata.FilterMetadata[LoadKind]
	if !ok {
		lb.Metadata.FilterMetadata[LoadKind] = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		lb.Metadata.FilterMetadata[LoadKind].Fields[LoadKind] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: load}}
		return
	}
	s.Fields[LoadKind].GetKind().(*structpb.Value_NumberValue).NumberValue += load
}
