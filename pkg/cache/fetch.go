package cache

import (
	"fmt"
	"sort"
	"strconv"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/miekg/xds/pkg/log"
	"github.com/miekg/xds/pkg/resource"
)

// Fetch fetches cluster data from the cluster. Here we probably deviate from the spec, as empty versions are allowed and we
// will return the full list again. For versioning we use the highest version we see in the cache and use that as the version
// in the reply.
func (c *Cluster) Fetch(req *discoverypb.DiscoveryRequest) (*discoverypb.DiscoveryResponse, error) {
	var resources []*any.Any
	if req.Node == nil {
		req.Node = &corepb.Node{Id: "ADS"}
	}

	switch req.TypeUrl {
	case resource.EndpointType:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)

		for _, n := range clusters {
			cluster, v := c.Retrieve(n)
			if cluster == nil {
				return nil, fmt.Errorf("cluster %q not found", n)
			}
			if v > version {
				version = v
			}
			endpoints := endpointpb.ClusterLoadAssignment(*(cluster.GetLoadAssignment()))
			data, err := MarshalResource(&endpoints)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		log.Debugf("Fetched endpoints (%d resources) for %q", len(resources), req.Node.Id)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

	case resource.ClusterType:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)

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
		log.Debugf("Fetched clusters (%d resources) for %q", len(resources), req.Node.Id)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
	case resource.ListenerType:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)

		for _, n := range clusters {
			cluster, v := c.Retrieve(n)
			if cluster == nil {
				return nil, fmt.Errorf("cluster %q not found", n)
			}
			if v > version {
				version = v
			}

			hcm := &httppb.HttpConnectionManager{
				RouteSpecifier: &httppb.HttpConnectionManager_Rds{
					Rds: &httppb.Rds{
						ConfigSource: &corepb.ConfigSource{
							ConfigSourceSpecifier: &corepb.ConfigSource_Ads{Ads: &corepb.AggregatedConfigSource{}},
						},
						RouteConfigName: cluster.Name,
					},
				},
			}
			hcmdata, _ := MarshalResource(hcm)
			lst := &listenerpb.Listener{
				Name: cluster.Name,
				ApiListener: &listenerpb.ApiListener{
					ApiListener: &any.Any{
						TypeUrl: resource.HttpConnManagerType,
						Value:   hcmdata,
					},
				},
			}
			data, err := MarshalResource(lst)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		log.Debugf("Fetched listeners (%d resources) for %q", len(resources), req.Node.Id)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
	case resource.RouteConfigType:
		sort.Strings(req.ResourceNames)
		clusters := req.ResourceNames
		if len(req.ResourceNames) == 0 {
			clusters = c.All()
		}
		version := uint64(0)

		for _, n := range clusters {
			cluster, v := c.Retrieve(n)
			if cluster == nil {
				return nil, fmt.Errorf("cluster %q not found", n)
			}
			if v > version {
				version = v
			}

			routec := &routepb.RouteConfiguration{
				Name: cluster.Name,
				VirtualHosts: []*routepb.VirtualHost{
					{
						Domains: []string{cluster.Name}, // cluster.Name, here??
						Routes: []*routepb.Route{
							{
								Match: &routepb.RouteMatch{PathSpecifier: &routepb.RouteMatch_Prefix{Prefix: ""}},
								Action: &routepb.Route_Route{
									Route: &routepb.RouteAction{
										ClusterSpecifier: &routepb.RouteAction_Cluster{Cluster: cluster.Name},
									},
								},
							},
						},
					},
				},
			}

			data, err := MarshalResource(routec)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		log.Debugf("Fetched routes (%d resources) for %q", len(resources), req.Node.Id)
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

	}
	return nil, fmt.Errorf("unrecognized/unsupported type %q:", req.TypeUrl)
}
