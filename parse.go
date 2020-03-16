package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/proto"
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
		cls = append(cls, pb)
	}
	return cls, nil
}
