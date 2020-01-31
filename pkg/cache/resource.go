// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package cache

import (
	"github.com/golang/protobuf/proto"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

// Resource is the base interface for the xDS payload.
type Resource interface {
	proto.Message
}

// Resource types in xDS v2.
const (
	apiTypePrefix       = "type.googleapis.com/envoy.api.v2."
	discoveryTypePrefix = "type.googleapis.com/envoy.service.discovery.v2."
	EndpointType        = apiTypePrefix + "ClusterLoadAssignment"
	ClusterType         = apiTypePrefix + "Cluster"

	// AnyType is used only by ADS
	AnyType = ""
)

// ResponseType enumeration of supported response types
type ResponseType int

const (
	Endpoint ResponseType = iota
	Cluster
	UnknownType // token to count the total number of supported types
)

// GetResponseType returns the enumeration for a valid xDS type URL
func GetResponseType(typeURL string) ResponseType {
	switch typeURL {
	case EndpointType:
		return Endpoint
	case ClusterType:
		return Cluster
	}
	return UnknownType
}

// GetResourceName returns the resource name for a valid xDS response type.
func GetResourceName(res Resource) string {
	switch v := res.(type) {
	case *endpointpb.ClusterLoadAssignment:
		return v.GetClusterName()
	case *clusterpb.Cluster:
		return v.GetName()
	default:
		return ""
	}
}

// MarshalResource converts the Resource to MarshaledResource
func MarshalResource(resource Resource) (MarshaledResource, error) {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(resource)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// GetResourceReferences returns the names for dependent resources (EDS cluster
// names for CDS, RDS routes names for LDS).
func GetResourceReferences(resources map[string]Resource) map[string]bool {
	out := make(map[string]bool)
	for _, res := range resources {
		if res == nil {
			continue
		}
		switch v := res.(type) {
		case *endpointpb.ClusterLoadAssignment:
			// no dependencies
		case *clusterpb.Cluster:
			// for EDS type, use cluster name or ServiceName override
			switch typ := v.ClusterDiscoveryType.(type) {
			case *clusterpb.Cluster_Type:
				if typ.Type == clusterpb.Cluster_EDS {
					if v.EdsClusterConfig != nil && v.EdsClusterConfig.ServiceName != "" {
						out[v.EdsClusterConfig.ServiceName] = true
					} else {
						out[v.Name] = true
					}
				}
			}
		}
	}
	return out
}
