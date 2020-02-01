# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

- xdsctl - cli to manipulate and list health and weight of endpoints and clusters.
- xds - management daemon that caches endpoints and clusters.

TLS is not implemented.

## Trying out

Start the server with `xds` and then use the client to connect to it with
`xdsctl -k -s 127.0.0.1:18000 ls`
