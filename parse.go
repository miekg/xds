package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/protobuf/proto"
)

func parseClusters(path string) ([]*endpointpb.ClusterLoadAssignment, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	cla := []*endpointpb.ClusterLoadAssignment{}
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
		pb := &endpointpb.ClusterLoadAssignment{}
		if err := proto.UnmarshalText(string(data), pb); err != nil {
			return nil, err
		}
		// suffix and prefix check, now the middle is the cluster name
		name := f.Name()[8 : len(f.Name())-7]
		if name != pb.GetClusterName() {
			return nil, fmt.Errorf("cluster name %q does not match file: %q: %s", pb.GetClusterName(), name, f.Name())
		}
		cla = append(cla, pb)
	}
	return cla, nil
}
