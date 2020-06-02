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

// Package resource creates xDS resources
package resource

// Resource types in xDS.
const (
	ClusterType     = "type.googleapis.com/envoy.api.v2.Cluster"
	EndpointType    = "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment"
	ListenerType    = "type.googleapis.com/envoy.api.v2.Listener"
	RouteConfigType = "type.googleapis.com/envoy.api.v2.RouteConfiguration"

	// These types need to be removed, they seem to not serve any purpose other than creating confusion.
	ClusterType3  = "type.googleapis.com/envoy.config.cluster.v3.Cluster"
	EndpointType3 = "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"

	HttpConnManagerType = "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager"

	// AnyType is used only by ADS
	AnyType = ""
)

// MakeCluster create a clusterpb.Cluster.
/*
func MakeCluster(name string) *clusterpb.Cluster {
	return &clusterpb.Cluster{
		Name:                 name,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_EDS},
		EdsClusterConfig: &clusterpb.Cluster_EdsClusterConfig{
			EdsConfig: &corepb.ConfigSource{
				ConfigSourceSpecifier: &corepb.ConfigSource_Ads{
					Ads: &corepb.AggregatedConfigSource{},
				},
			},
		},
	}
}
*/
