package cache

import (
	"fmt"
	"sort"
	"strconv"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	routepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	httppb2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	listenerpb2 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v2"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/miekg/xds/pkg/resource"
)

// Fetch fetches cluster data from the cluster. Here we probably deviate from the spec, as empty versions are allowed and we
// will return the full list again. For versioning we use the highest version we see in the cache and use that as the version
// in the reply.
func (c *Cluster) Fetch(req *xdspb2.DiscoveryRequest) (*xdspb2.DiscoveryResponse, error) {
	var resources []*any.Any
	if req.Node == nil {
		req.Node = &corepb2.Node{Id: "ADS"}
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
			endpoints := xdspb2.ClusterLoadAssignment(*(cluster.GetLoadAssignment()))
			data, err := MarshalResource(&endpoints)
			if err != nil {
				return nil, err
			}
			resources = append(resources, &any.Any{TypeUrl: req.TypeUrl, Value: data})
		}
		versionInfo := strconv.FormatUint(version, 10)
		return &xdspb2.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

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
		return &xdspb2.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
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

			hcm := &httppb2.HttpConnectionManager{
				RouteSpecifier: &httppb2.HttpConnectionManager_Rds{
					Rds: &httppb2.Rds{
						ConfigSource: &corepb2.ConfigSource{
							ConfigSourceSpecifier: &corepb2.ConfigSource_Ads{Ads: &corepb2.AggregatedConfigSource{}},
						},
						RouteConfigName: cluster.Name,
					},
				},
			}
			hcmdata, _ := MarshalResource(hcm)
			lst := &xdspb2.Listener{
				Name: cluster.Name,
				ApiListener: &listenerpb2.ApiListener{
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
		return &xdspb2.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
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

			routec := &xdspb2.RouteConfiguration{
				Name: cluster.Name,
				VirtualHosts: []*routepb2.VirtualHost{
					{
						Domains: []string{cluster.Name}, // cluster.Name, here??
						Routes: []*routepb2.Route{
							{
								Match: &routepb2.RouteMatch{PathSpecifier: &routepb2.RouteMatch_Prefix{Prefix: ""}},
								Action: &routepb2.Route_Route{
									Route: &routepb2.RouteAction{
										ClusterSpecifier: &routepb2.RouteAction_Cluster{Cluster: cluster.Name},
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
		return &xdspb2.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

	}
	return nil, fmt.Errorf("unrecognized/unsupported type %q:", req.TypeUrl)
}
