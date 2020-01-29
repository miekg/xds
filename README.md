# xdsctl

## Name

xdsctl - communicate with a xDS endpoint.

The are several commands implemented, just look at the help output of xdsctl (which should be fairly
complete).

To keep things relatively simple *all* command will result in sending a DiscoveryRequest (reads) or
a DiscoveryResponse to the xDS capable endpoint. The v2 API of Envoy's xDS is currently implemented.

## See Also

<https://github.com/miekg/xdsd> is an xDS endpoint that implements a sane set of action upon
receiving these requests.

## Bugs

What if you drain a cluster and then a new healthy end point is added?
