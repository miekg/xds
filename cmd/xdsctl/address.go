package main

import (
	"net"
	"strconv"

	corepb2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	corepb3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// Convert a corepb2.Address to a corepb3 address.
func addressToV3(a *corepb2.Address) *corepb3.Address {
	port := a.Address.(*corepb2.Address_SocketAddress).SocketAddress.PortSpecifier.(*corepb2.SocketAddress_PortValue).PortValue
	addr := a.Address.(*corepb2.Address_SocketAddress).SocketAddress.Address
	return &corepb3.Address{
		Address: &corepb3.Address_SocketAddress{
			&corepb3.SocketAddress{
				Address:       addr,
				PortSpecifier: &corepb3.SocketAddress_PortValue{port},
			},
		},
	}
}

func coreAddressToAddr(sa *corepb2.Address_SocketAddress) string {
	addr := sa.SocketAddress.Address

	port, ok := sa.SocketAddress.PortSpecifier.(*corepb2.SocketAddress_PortValue)
	if !ok {
		return addr
	}
	return net.JoinHostPort(addr, strconv.FormatUint(uint64(port.PortValue), 10))
}
