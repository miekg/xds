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
	"github.com/miekg/xds/pkg/resource"
	"google.golang.org/protobuf/types/known/anypb"
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
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil

	case resource.ClusterType, resource.ClusterType3:
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
		return &discoverypb.DiscoveryResponse{VersionInfo: versionInfo, Resources: resources, TypeUrl: req.TypeUrl}, nil
		// gRPC uses these types to create a server config
		// in grpcLB this was returned via a DNS TXT record.
	case resource.ListenerType:
		hcm := &httppb.HttpConnectionManager{
			RouteSpecifier: &httppb.HttpConnectionManager_Rds{
				Rds: &httppb.Rds{
					ConfigSource: &corepb.ConfigSource{
						ConfigSourceSpecifier: &corepb.ConfigSource_Ads{Ads: &corepb.AggregatedConfigSource{}},
					},
					RouteConfigName: "helloworld", // <-- also cluster name?!
				},
			},
		}
		hcmdata, _ := MarshalResource(hcm)
		lst := &listenerpb.Listener{
			Name: "helloworld", // <-- cluster name!
			ApiListener: &listenerpb.ApiListener{
				ApiListener: &anypb.Any{
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
		return &discoverypb.DiscoveryResponse{VersionInfo: "1", Resources: resources, TypeUrl: req.TypeUrl}, nil
	case resource.RouteConfigType:
		routec := &routepb.RouteConfiguration{
			Name: "helloworld",
			VirtualHosts: []*routepb.VirtualHost{
				{
					Domains: []string{"helloworld"},
					Routes: []*routepb.Route{
						{
							Match: &routepb.RouteMatch{PathSpecifier: &routepb.RouteMatch_Prefix{Prefix: ""}},
							Action: &routepb.Route_Route{
								Route: &routepb.RouteAction{
									ClusterSpecifier: &routepb.RouteAction_Cluster{Cluster: "helloworld"},
								},
							},
						},
					},
				},
				{
					Domains: []string{"helloworld"},
					Routes: []*routepb.Route{
						{
							Match: &routepb.RouteMatch{PathSpecifier: &routepb.RouteMatch_Prefix{Prefix: ""}},
							Action: &routepb.Route_Route{
								Route: &routepb.RouteAction{
									ClusterSpecifier: &routepb.RouteAction_Cluster{Cluster: "helloworld"},
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
		return &discoverypb.DiscoveryResponse{VersionInfo: "1", Resources: resources, TypeUrl: req.TypeUrl}, nil

	}
	return nil, fmt.Errorf("unrecognized/unsupported type %q:", req.TypeUrl)
}
