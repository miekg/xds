package cache2

import (
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/proto"
)

// Resource types in xDS v3.
const (
	discoveryTypePrefix = "type.googleapis.com/envoy.service.discovery.v3."
	apiTypePrefix       = "type.googleapis.com/envoy.config."
	EndpointType        = apiTypePrefix + "endpoint.v3.ClusterLoadAssignment"
	ClusterType         = apiTypePrefix + "cluster.v3.Cluster"

	// AnyType is used only by ADS
	AnyType = ""
)

type Response struct {
	*discoverypb.DiscoveryResponse
	version string
}

type MarshaledResource = []byte

// MarshalResource converts the Resource to MarshaledResource
func MarshalResource(resource proto.Message) (MarshaledResource, error) {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(resource)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
