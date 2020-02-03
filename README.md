# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

 *  xdsctl - cli to manipulate and list health and weight of endpoints and clusters.

 *  xds - management daemon that caches endpoints and clusters.

TLS is not implemented.

Note that this implements the v3 xDS API, but at the time (Jan 2020) that was still in development,
so it may deviate from the final version. If so, it is expected this repo will change to reflect to
actual, implemented v3 API.

## Trying out

Start the server with `xds` and then use the client to connect to it with `xdsctl -k -s
127.0.0.1:18000 ls`

## xds

 *  Adds clusters via a text protobuf on startup, after reading this in the version will be set to
 *  v1.

 *  Allow xdsctl to set weights/statuses, up the version, etc. Note that weight updates don't have a
    protocol defined (i.e. no WeightReportingService)

 *  Any update of the cluster means moving up a version.

 *  Do both xDS (fetches) and ADS

 *  When xds starts up, files adhering to this glob "cluster.*.textpb" will be parsed as
    ClusterLoadAssigment protobuffer in text format. These define the set of cluster we know about.
    Note: this is in effect the "admin interface", until we figure out how it should look.
    The wildcard should match the name of cluster being defined in the protobuf.
