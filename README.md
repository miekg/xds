# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

 *  xds - management daemon that caches endpoints and clusters and hands them out using xDS and ADS.

 *  xdsctl - cli to manipulate and list details of endpoints and clusters.

TLS is not implemented (yet). Note that this implements the v3 xDS API, Envoy works with this API as
well (by having a few parts still in the v2 format).

xdsctl uses xDS to manipulate the cluster info stored. All other users that read from it must use
ADS. Every 10 seconds `xds` will send out an update (if there are changes) to all connected clients.

Load reporting (LRS) is supported, only one dimensions though: queries. The xds management server
uses the reported load to adjust the weights of the endpoints.

## Trying out

Build the server and clients:

 *  server: `go build`

 *  client: `cd cmd/xdsctl; go build`

Start the server with `xds` and then use the client to connect to it with `xdsctl -k -s
127.0.0.1:18000 ls`. When starting up `xds` will read files `cluster.*.textpb` that contain clusters
to use. This will continue during the runtime of the process; new clusters - if found - will be
added. Removal is not implemented (yet).

Both xDS and ADS are implemented by `xds`. For xDS we force the types to the v3 protos. For ADS (and
to make Envoy happy) we support also v2 - this may not be interely up to specification though).

The `envoy-bootstrap.yaml` can be used to point Envoy to the xds control plane - note this only
gives envoy CDS/EDS responses (via ADS), so no listeners nor routes. Envoy can be downloaded from
<https://tetrate.bintray.com/getenvoy/>.

CoreDNS (with the *traffic* plugin compiled in), can be started with the Corefile specified to get
DNS responses out of xds. CoreDNS can be found at <https://coredns.io>

## xds

 *  Adds clusters via a text protobuf on startup, after reading this in the version will be set to
    v1 for those.

 *  When xds starts up, files adhering to this glob "cluster.*.textpb" will be parsed as
    Cluster protobuffer in text format. These define the set of clusters we know about.
    Note: this is in effect the "admin interface", until we figure out how it should look. The
    wildcard should match the name of cluster being defined in the protobuf.

See cmd/xdsctl/README.md for how to use the CLI.

## Bugs

What if you drain a cluster and then a new healthy end point is added? This new endpoint will get
health checked and possibly be set health, meaning *all* traffic will flow to this one endpoint.
