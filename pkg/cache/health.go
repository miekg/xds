package cache

import (
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	healthpb "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
)

// SetHealth sets the health for clusters and or endpoints.
func (c *Cluster) SetHealth(req *healthpb.EndpointHealthResponse) (*healthpb.HealthCheckSpecifier, error) {
	toChange := make([]string, len(req.EndpointsHealth))
	health := make([]corepb.HealthStatus, len(req.EndpointsHealth))
	for i, ep := range req.EndpointsHealth {
		toChange[i] = ep.GetEndpoint().GetAddress().GetSocketAddress().String()
		health[i] = ep.HealthStatus
	}

	// we lack a cluster name, so we iterate over *all* clusters that have this endpoint and set it's health,
	// not sure if this is how it is supposed to work.
	all := c.All()
	for _, name := range all {
		cluster, _ := c.Retrieve(name)

		done := false
		endpoints := cluster.GetLoadAssignment()
		for _, ep := range endpoints.Endpoints {
			for _, lb := range ep.GetLbEndpoints() {
				epa := lb.GetEndpoint().GetAddress().GetSocketAddress()
				for j, sa := range toChange {
					if sa == epa.String() {
						if lb.HealthStatus != health[j] {
							lb.HealthStatus = health[j]
							done = true
						}
					}
				}
			}
		}
		if done {
			// we've updated something, write it back to the cache.
			c.Insert(cluster)
		}
	}

	return &healthpb.HealthCheckSpecifier{}, nil
}
