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
	mu      sync.RWMutex
	c       map[string]*endpointpb.ClusterLoadAssignment
	version uint64 // if anything changes this gets a new version.
}

func New() *Cluster {
	return &Cluster{c: make(map[string]*endpointpb.ClusterLoadAssignment)}
}

func (c *Cluster) Insert(ep *endpointpb.ClusterLoadAssignment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.version += 1
	c.c[ep.GetClusterName()] = ep
}

func (c *Cluster) Retrieve(name string) (*endpointpb.ClusterLoadAssignment, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ep, ok := c.c[name]
	if !ok {
		return nil, 0
	}
	return ep, c.version
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

func (c *Cluster) Version() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.version
}
