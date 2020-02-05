# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

 *  xdsctl - cli to manipulate and list health and weight of endpoints and clusters.

 *  xds - management daemon that caches endpoints and clusters.

TLS is not implemented (yet).

Note that this implements the v3 xDS API, Envoy works with this API as well. There is an admin
interface specified, that uses the same protobufs (DiscoveryResponse) on a different endpoint.

## Trying out

Build the server and clients:

* server: `go build`
* client: `cd cmd/xdsctl; go build`

Start the server with `xds` and then use the client to connect to it with `xdsctl -k -s
127.0.0.1:18000 ls`. When starting up `xds` will read files `cluster.*.textpb` that contain
clusters to use on startup.

Both xDS and ADS are implemented by `xds`.

## xds

 *  Adds clusters via a text protobuf on startup, after reading this in the version will be set to
 *  v1 for those.

 *  Allow xdsctl to set weights/statuses, up the version, etc. Some things have not been implemented
    yet, mostly because no protocol has been defined yet (i.e weights).

 *  When xds starts up, files adhering to this glob "cluster.*.textpb" will be parsed as
    ClusterLoadAssigment protobuffer in text format. These define the set of cluster we know about.
    Note: this is in effect the "admin interface", until we figure out how it should look.
    The wildcard should match the name of cluster being defined in the protobuf.

## Bugs

What if you drain a cluster and then a new healthy end point is added? This new endpoint will get
health checked and possiby be set health, meaning *all* traffic will flow to this one endpoint.

## TODO

* new clusters - send updates with ADS ? Double check with Envoy

## Protocol Notes

per stream (node-id) need to keep track of this: or do this by default:

  Note that once a stream has entered wildcard mode for a given resource type, there is no way to
  change the stream out of wildcard mode; resource names specified in any subsequent request on the
  stream will be ignored.
