package cache

import (
	xdspb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/gogo/protobuf/proto"
)

type Response struct {
	*xdspb2.DiscoveryResponse
	version string
}

type MarshaledResource = []byte

// MarshalResource converts the Resource to a MarshaledResource.
func MarshalResource(resource proto.Message) (MarshaledResource, error) {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(resource)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
