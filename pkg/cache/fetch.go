package cache

import (
	"fmt"
	"sort"
	"strconv"

	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/miekg/xds/pkg/resource"
)

// Fetch fetches cluster data from the cluster. Here we probably deviate from the spec, as empty versions are allowed and we
// will return the full list again. For versioning we use the highest version we see in the cache and use that as the version
// in the reply.
func (c *Cluster) Fetch(req *discoverypb.DiscoveryRequest) (*discoverypb.DiscoveryResponse, error) {
	var resources []*any.Any

	switch req.TypeUrl {
	case resource.EndpointType, resource.EndpointType3:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)
		for _, n := range clusters {
			cla, v := c.Retrieve(n)
			if cla == nil {
				return nil, fmt.Errorf("cluster %q not found", n)
			}
			if v > version {
				version = v
			}
			data, err := MarshalResource(cla)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

	case resource.ClusterType, resource.ClusterType3:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)
		// As we only store ClusterLoadAssignments, we need to create a cluster response.
		for _, n := range clusters {
			cluster, v := c.Retrieve(n)
			if cluster == nil {
				return nil, fmt.Errorf("cluster %q not found", n)
			}
			if v > version {
				version = v
			}
			data, err := MarshalResource(cluster)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
	}
	return nil, fmt.Errorf("unrecognized/unsupported type %q:", req.TypeUrl)
}
