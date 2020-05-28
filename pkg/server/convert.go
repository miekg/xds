package server

import (
	"log"

	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/proto"
)

// DiscoveryResponseToV2 converts a v3 proto struct to a v2 one.
func DiscoveryResponseToV2(r *discoverypb.DiscoveryResponse) *xdspb2.DiscoveryResponse {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(r)

	err = err
	x := &xdspb2.DiscoveryResponse{}
	if err := proto.Unmarshal(b.Bytes(), x); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}

	return x
}

// DiscoveryRequestToV3 converts a v2 proto struct to a v3 one.
func DiscoveryRequestToV3(r *xdspb2.DiscoveryRequest) *discoverypb.DiscoveryRequest {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(r)

	err = err
	x := &discoverypb.DiscoveryRequest{}
	if err := proto.Unmarshal(b.Bytes(), x); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}

	return x
}
