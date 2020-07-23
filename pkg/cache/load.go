package cache

import (
	"strings"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
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
			log.Debugf("Load report for unknown cluster %s", clusterStats.ClusterName)
			continue
		}

		// This is our hack to set the weights via load reporting.
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
			// doing this string slice trick is (mem) wasteful, but ease the code (a little). Should
			// be removed/improved. TODO(miek).
			locs := []string{}
			loc := upstreamStats.GetLocality()
			if x := loc.GetRegion(); x != "" {
				locs = append(locs, x)
			}
			if x := loc.GetZone(); x != "" {
				locs = append(locs, x)
			}
			if x := loc.GetSubZone(); x != "" {
				locs = append(locs, x)
			}
			where := strings.TrimSpace(strings.Join(locs, "/")) // this is also the metadata key for this load report in this cluster

			// grpc reports: TotalSuccessfulRequests
			totalSuccessLoad := upstreamStats.GetTotalSuccessfulRequests()
			// check if any of the endpoints match the locality, if so, then set the load
			// in the cluster's metadata
			for _, ep := range endpoints.Endpoints {
				locs := []string{}
				loc := ep.GetLocality()
				if x := loc.GetRegion(); x != "" {
					locs = append(locs, x)
				}
				if x := loc.GetZone(); x != "" {
					locs = append(locs, x)
				}
				if x := loc.GetSubZone(); x != "" {
					locs = append(locs, x)
				}
				ewhere := strings.TrimSpace(strings.Join(locs, "/")) // this is also the metadata key for this load report in this cluster
				if ewhere == where {
					SetLoadInMetadata(cl, where, totalSuccessLoad)
					log.Debugf("Load report for %s, reporting %d for locality %s", cl.Name, totalSuccessLoad, where)
					done = true
				}
			}
		}
		if done {
			// we've updated something, write it back to the cache.
			c.InsertWithoutVersionUpdate(cl)
			continue
		}
		log.Debugf("Load report for unknown locality in cluster %s", clusterStats.ClusterName)
	}
	// if there wasn't an actual load report this was the initial ping that load "are coming", in that case
	// node Id contains the cluster we're interested in, so put that in the cluster slice.
	if len(clusters) == 0 {
		clusters = []string{req.Node.Id}
	}
	return &loadpb2.LoadStatsResponse{
		Clusters:              clusters,
		LoadReportingInterval: &duration.Duration{Seconds: 2},
		// ReportEndpointGranularity: true, // we use the locality, endpoint load isn't implemented yet...
	}, nil
}

// LoadFromMetada returns the load as reported in the metadata of the endpoint.
func LoadFromMetadata(cl *xdspb2.Cluster, locality string) uint64 {
	if cl.Metadata == nil {
		return 0
	}
	s, ok := cl.Metadata.FilterMetadata[LoadKind] // we store the load here
	if !ok {
		return 0
	}
	if s.Fields == nil {
		return 0
	}
	sv := s.Fields[LoadKind+locality] // 'load' again, because nested maps
	return uint64(sv.GetNumberValue())
}

// SetLoadInMetadata adds load to the metadata of the LbEndpoint.
func SetLoadInMetadata(cl *xdspb2.Cluster, locality string, load uint64) {
	if cl.Metadata == nil {
		cl.Metadata = new(corepb2.Metadata)
	}
	if cl.Metadata.FilterMetadata == nil {
		cl.Metadata.FilterMetadata = map[string]*structpb.Struct{}
	}
	_, ok := cl.Metadata.FilterMetadata[LoadKind]
	if !ok {
		cl.Metadata.FilterMetadata[LoadKind] = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if _, ok := cl.Metadata.FilterMetadata[LoadKind].Fields[LoadKind+locality]; !ok {
		cl.Metadata.FilterMetadata[LoadKind].Fields[LoadKind+locality] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(0)}}
	}
	cl.Metadata.FilterMetadata[LoadKind].Fields[LoadKind+locality].GetKind().(*structpb.Value_NumberValue).NumberValue += float64(load)
}

func TotalLoadFromMetadata(cl *xdspb2.Cluster) uint64 {
	if cl.Metadata == nil {
		return 0
	}
	s, ok := cl.Metadata.FilterMetadata[LoadKind] // we store the load here
	if !ok {
		return 0
	}
	if s.Fields == nil {
		return 0
	}
	load := uint64(0)
	for _, sv := range s.Fields {
		load += uint64(sv.GetNumberValue())
	}
	return load

}
