# xdsctl

## Name

xdsctl - communicate with a xDS endpoint.

The are several commands implemented, just look at the help output of xdsctl (which should be fairly
complete).

We use xDS (v3) to extract discovery data from the management server. Health
reporting (and setting endpoint state) is done via the Health Disovery Service
(HealthCheckRequestOrEndpointHealthResponse), where we send EndpointHealthResponses to it.

The "admin" site of this tool (add, rm) isn't implemented yet, because I can't find the protobufs
that I need to implement this; this can be worked around by using CDS and or EDS to send a
"discovery response" that's seen as a cue to add or remove. However distinct protos (which could be
very similar to DiscoveryRequest) would work better.

## Bugs

What if you drain a cluster and then a new healthy end point is added?

## TODO

* Add the version from the response?
