package cache

import (
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/gogo/protobuf/proto"
)

type Response struct {
	*discoverypb.DiscoveryResponse
	version string
}

type MarshaledResource = []byte

// MarshalResource converts the Resource to MarshaledResource
func MarshalResource(resource proto.Message) (MarshaledResource, error) {
	b := proto.NewBuffer(nil)
	b.SetDeterministic(true)
	err := b.Marshal(resource)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
