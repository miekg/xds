package cache

import (
	"sort"
	"sync"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	deep "github.com/mitchellh/copystructure"
)

// Clusters holds the current clusters. For each cluster we only keep the ClusterLoadAssignments, for ClusterType
// queries we will create a reply on-the-fly. We don't care about node-id's, but we do check the version of the
// incoming reply to see if we have a newer one.
type Cluster struct {
	mu      sync.RWMutex
	c       map[string]*xdspb2.Cluster
	version uint64 // if anything changes this gets a new version.
}

func New() *Cluster {
	return &Cluster{c: make(map[string]*xdspb2.Cluster)}
}

func (c *Cluster) Insert(ep *xdspb2.Cluster) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.version += 1
	c.c[ep.GetName()] = ep
}

func (c *Cluster) InsertWithoutVersionUpdate(ep *xdspb2.Cluster) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c[ep.GetName()] = ep
}

func (c *Cluster) Retrieve(name string) (*xdspb2.Cluster, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ep, ok := c.c[name]
	if !ok {
		return nil, 0
	}
	dc, _ := deep.Copy(ep)
	return dc.(*xdspb2.Cluster), c.version
}

// All returns all cluster names in alphabetical order available in the cache.
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

// Version returns the version of the cluster.
func (c *Cluster) Version() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.version
}

const (
	WeightKind = "weight" // Key name in metadata where the weight is stored.
	LoadKind   = "load"   // Key name in the metadata where the load is stored.
	HashKind   = "hash"   // hash of the textpb cluster definition.
)
