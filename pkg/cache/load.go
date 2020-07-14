package cache

import (
	"fmt"

	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	edspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	loadpb2 "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
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
		for _, upstreamStats := range clusterStats.UpstreamLocalityStats {
			for _, endpointStats := range upstreamStats.UpstreamEndpointStats {
				fmt.Printf("%s\n", endpointStats.Address)
				fmt.Printf("%d\n", endpointStats.TotalIssuedRequests)
			}
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
	}
	s.Fields["LOAD"].GetKind().(*structpb.Value_NumberValue).NumberValue += load
}
