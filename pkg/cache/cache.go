package cache

import (
	"sort"
	"sync"

	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

// Clusters holds the current clusters. For each cluster we only keep the ClusterLoadAssignments, for ClusterType
// queries we will create a reply on-the-fly. We don't care about node-id's, but we do check the version of the
// incoming reply to see if we have a newer one.
type Cluster struct {
	mu sync.RWMutex
	c  map[string]Assignment
}

// Assignment holds the versioned ClusterLoadAssignment. Version will only be 0 when there isn't any data.
type Assignment struct {
	*endpointpb.ClusterLoadAssignment
	Version uint64
}

func New() *Cluster {
	return &Cluster{c: make(map[string]Assignment)}
}

func (c *Cluster) Insert(ep *endpointpb.ClusterLoadAssignment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	a, ok := c.c[ep.GetClusterName()]
	if !ok {
		c.c[ep.GetClusterName()] = Assignment{ep, 1}
		return
	}
	v := a.Version + 1
	c.c[ep.GetClusterName()] = Assignment{ep, v}
}

func (c *Cluster) Retrieve(name string) (*endpointpb.ClusterLoadAssignment, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	a, ok := c.c[name]
	if !ok {
		return nil, 0
	}
	return a.ClusterLoadAssignment, a.Version
}

// All returns all cluster names available in the cache. The returns list will be alphabetically sorted.
func (c *Cluster) All() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.c))
	for k, _ := range c.c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
