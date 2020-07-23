package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/miekg/xds/pkg/cache"
)

func parseClusters(path string) ([]*xdspb2.Cluster, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	cls := []*xdspb2.Cluster{}
	for _, f := range dir {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) != ".textpb" {
			continue
		}
		if !strings.HasPrefix(f.Name(), "cluster.") {
			continue
		}
		data, err := ioutil.ReadFile(filepath.Join(path, f.Name()))
		if err != nil {
			return nil, err
		}

		// suffix and prefix check, now the middle is the cluster name
		name := f.Name()[8 : len(f.Name())-7]

		pb := &xdspb2.Cluster{}
		if err := proto.UnmarshalText(string(data), pb); err != nil {
			return nil, fmt.Errorf("cluster %q: %s", name, err)
		}
		if name != pb.GetName() {
			return nil, fmt.Errorf("cluster name %q does not match file: %q: %s", pb.GetName(), name, f.Name())
		}
		// some sanity checks
		if pb.GetType() != xdspb2.Cluster_EDS {
			return nil, fmt.Errorf("cluster %q must have discovery type set to EDS", name)
		}
		hcs := pb.GetHealthChecks()
		if len(hcs) == 0 {
			return nil, fmt.Errorf("cluster %q must have health checks", name)
		}
		for _, hc := range hcs {
			setDurationIfNil(hc.Timeout, 5*time.Second, fmt.Sprintf("Cluster %q, setting %s to", name, "Timeout"))
			setDurationIfNil(hc.Interval, 10*time.Second, fmt.Sprintf("Cluster %q, setting %s to", name, "Internval"))
			setDurationIfNil(hc.InitialJitter, 2*time.Second, fmt.Sprintf("Cluster %q, setting %s to", name, "InitialJitter"))
			setDurationIfNil(hc.IntervalJitter, 1*time.Second, fmt.Sprintf("Cluster %q, setting %s to", name, "IntervalJitter"))
		}
		pb.EdsClusterConfig = &xdspb2.Cluster_EdsClusterConfig{
			EdsConfig: &corepb2.ConfigSource{ConfigSourceSpecifier: &corepb2.ConfigSource_Ads{Ads: &corepb2.AggregatedConfigSource{}}},
		}

		// Now we're fixing up clusters, by setting some missing value and defaulting settings (mostly durations) that may be left out.
		endpoints := pb.GetLoadAssignment()

		// If the endpoints cluster name if not set, set it to the cluster name
		if endpoints.ClusterName != pb.GetName() {
			endpoints.ClusterName = pb.GetName()
		}

		// hash the file and set in the metadata.
		h := sha1.New()
		h.Write(data)
		bs := h.Sum(nil)
		cache.SetHashInMetadata(pb, fmt.Sprintf("%x", bs))

		cls = append(cls, pb)
	}
	return cls, nil
}

func setDurationIfNil(a *duration.Duration, v time.Duration, msg string) {
	if a != nil {
		return
	}
	a = &duration.Duration{Seconds: int64(v / time.Second)} // skip Nanos
}
