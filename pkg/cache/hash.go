package cache

import (
	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	structpb "github.com/golang/protobuf/ptypes/struct"
)

// HashFromMetada returns the hash from the metadata.
func HashFromMetadata(cl *xdspb2.Cluster) string {
	if cl.Metadata == nil {
		return ""
	}
	s, ok := cl.Metadata.FilterMetadata[HashKind] // we store the load here
	if !ok {
		return ""
	}
	if s.Fields == nil {
		return ""
	}
	sv := s.Fields[HashKind]
	return sv.GetStringValue()
}

// SetHashInMetadata set the hash in the metadata.
func SetHashInMetadata(cl *xdspb2.Cluster, hash string) {
	if cl.Metadata == nil {
		cl.Metadata = new(corepb2.Metadata)
	}
	if cl.Metadata.FilterMetadata == nil {
		cl.Metadata.FilterMetadata = map[string]*structpb.Struct{}
	}
	_, ok := cl.Metadata.FilterMetadata[HashKind]
	if !ok {
		cl.Metadata.FilterMetadata[HashKind] = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if _, ok := cl.Metadata.FilterMetadata[HashKind].Fields[HashKind]; !ok {
		cl.Metadata.FilterMetadata[HashKind].Fields[HashKind] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: ""}}
	}
	cl.Metadata.FilterMetadata[HashKind].Fields[HashKind].GetKind().(*structpb.Value_StringValue).StringValue = hash
}
