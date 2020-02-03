package cache

import (
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
)

// SetHealth sets the health for clusters and or endpoints.
func (c *Cluster) SetHealth(req *healthpb.EndpointHealthResponse) (*healthpb.HealthCheckSpecifier, error) {
	// we lack a cluster name, so we iterate over *all* clusters that have this endpoint and set it's health,
	// not sure if this is how it is supposed to work.
	all := c.All()
	endpoints := make([]*endpointpb.ClusterLoadAssignment, len(all))
	for i, cluster := range all {
		endpoints[i], _ = c.Retrieve(cluster)
	}

	toChange := make([]string, len(req.EndpointsHealth))
	health := make([]corepb.HealthStatus, len(req.EndpointsHealth))
	for i, ep := range req.EndpointsHealth {
		toChange[i] = ep.GetEndpoint().GetAddress().GetSocketAddress().String()
		health[i] = ep.HealthStatus
	}

	for i := 0; i < len(all); i++ {
		done := false
		for _, ep := range endpoints[i].Endpoints {
			for _, lb := range ep.GetLbEndpoints() {
				epa := lb.GetEndpoint().GetAddress().GetSocketAddress()
				for j, sa := range toChange {
					if sa == epa.String() { // strings...
						if lb.HealthStatus != health[j] {
							lb.HealthStatus = health[j]
							done = true
						}
					}
				}
			}
		}
		if done {
			// we updates something, write it back to the cache.
			c.Insert(endpoints[i])
		}
	}

	return &healthpb.HealthCheckSpecifier{}, nil
}
