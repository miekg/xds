# xds

xDS is Envoy's discovery protocol. This repo contains xDS related utilities - included are:

 *  xds - management daemon that caches endpoints and clusters and hands them out using xDS and ADS.

 *  xdsctl - cli to manipulate and list details of endpoints and clusters.

TLS is not implemented (yet). Note that this implements the v2 xDS API, Envoy works with this API
as well.

There is an admin interface specified, that uses the same protobufs (DiscoveryResponse) on a
different endpoint. xdsctl uses xDS to manipulate the cluster info stored. All other users that read
from it must use ADS. Every 10 seconds `xds` will send out an update (if there are changes) to all
connected clients.

THIS IS A PROTOTYPE IMPLEMENTATION. It may get extended to actual production quality at some point.

## Trying out

Build the server and clients:

 *  server: `go build`

 *  client: `cd cmd/xdsctl; go build`

 *  helloworld client and server: `cd helloworld/{client,server}; go build`

Start the server with `xds` and then use the client to connect to it with `xdsctl -k -s
127.0.0.1:18000 ls`. When starting up `xds` will read files `cluster.*.textpb` that contain clusters
to use. This will continue during the runtime of the process; new clusters - if found - will be
added. Removal is not implemented (yet).

Both xDS and ADS are implemented by `xds`.

The `envoy-bootstrap.yaml` can be used to point Envoy to the xds control plane - note this only
gives envoy CDS/EDS responses (via ADS), so no listeners nor routes. Envoy can be downloaded from
<https://tetrate.bintray.com/getenvoy/>.

CoreDNS (with the *traffic* plugin compiled in), can be started with the Corefile specified to get
DNS responses out of xds. CoreDNS can be found at <https://coredns.io>

## xds

 *  Adds clusters via a text protobuf on startup, after reading this in the version will be set to
    v1 for those.

 *  When xds starts up, files adhering to this glob "cluster.*.textpb" will be parsed as
    Cluster protocol buffer in text format. These define the set of clusters we know about.
    Note: this is in effect the "admin interface", until we figure out how it should look. The
    wildcard should match the name of cluster being defined in the protobuf.

See cmd/xdsctl/README.md for how to use the CLI.

In xds the following protocols have been implemented:

* xDS - Envoy's configuration and discovery protocol (includes LDS, RDS, EDS and CDS)
* LRS - load reporting (also from Envoy) - not implemented yet.

For debugging add:

~~~ sh
export RPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
~~~

For helping the xds client bootstrap set: `export GRPC_XDS_BOOTSTRAP=./boostrap.json`

## Usage

Start the management server, the servers and then the client:

~~~
% ./xds -debug
~~~

Servers (these match the endpoints as defined in the `cluster.hellowold.textpb` file.

~~~
% ./helloworld/server/server -addr 127.0.1.1:50051 &
% ./helloworld/server/server -addr 127.0.0.1:50051 &
~~~

And then query:

~~~
% ./helloworld/client/client -addr xds:///helloworld
~~~

Note you can specify a DNS server to use, but then the client will *also* do DNS looks up and you
get a weird mix of grpclb and xDS behavior:

~~~
% ./helloworld/client/client -addr dns://127.0.0.1:1053/helloworld.lb.example.org:50501
~~~

## Load Reporting

Load reporting is supported via LRS. However to save the reporting load back into the cluster, we
use the metadata field of of the `*endpointdb2.LbEndpoint` where we store this value. This allows
`xdscli` to extract it from the management server without adding new endpoints.

## Admin Interface

On a different port an admin interface is running, this is basically shuffing the same protos over
the wire, but in the opposite direction. This is currently only used to adjust the weight of
clusters. In this case, first a cluster is retrieved via the normal interface and the re-uploaded
with changed weights on the admin interface.

## TODO

* version per cluster; right now the version if global; if any cluster changes, the version is
  upped globaly.
* canceling watches and a lot more of this stuff

## Stuff Learned

* gRPC must see `load_balancing_weight`, otherwise it will silently drop the endpoints
* gRPC must have endpoints in different localities otherwise it will only use one?
