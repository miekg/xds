# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

- xdsctl - cli to manipulate and list health and weight of endpoints and clusters.
- xds - management daemon that caches endpoints and clusters.

TLS is not implemented.

Note that this implements the v3 xDS API, but at the time (Jan 2020) that was still in development,
so it may deviate from the final version. If so, it is expected this repo will change to reflect to
actual, implemented v3 API.

## Trying out

Start the server with `xds` and then use the client to connect to it with
`xdsctl -k -s 127.0.0.1:18000 ls`
