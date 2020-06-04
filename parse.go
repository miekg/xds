package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/duration"
)

func parseClusters(path string) ([]*clusterpb.Cluster, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	cls := []*clusterpb.Cluster{}
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

		pb := &clusterpb.Cluster{}
		if err := proto.UnmarshalText(string(data), pb); err != nil {
			return nil, fmt.Errorf("cluster %q: %s", name, err)
		}
		if name != pb.GetName() {
			return nil, fmt.Errorf("cluster name %q does not match file: %q: %s", pb.GetName(), name, f.Name())
		}
		// some sanity checks
		if pb.GetType() != clusterpb.Cluster_EDS {
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
		pb.EdsClusterConfig = &clusterpb.Cluster_EdsClusterConfig{
			EdsConfig: &corepb.ConfigSource{ConfigSourceSpecifier: &corepb.ConfigSource_Ads{Ads: &corepb.AggregatedConfigSource{}}},
		}

		// Now we're fixing up clusters, by setting some missing value and defaulting settings (mostly durations) that may be left out.
		endpoints := pb.GetLoadAssignment()

		// If the endpoints cluster name if not set, set it to the cluster name
		if endpoints.ClusterName != pb.GetName() {
			endpoints.ClusterName = pb.GetName()
		}

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
