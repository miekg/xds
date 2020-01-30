# xdsctl

## Name

xdsctl - communicate with a xDS endpoint.

The are several commands implemented, just look at the help output of xdsctl (which should be fairly
complete).

We use xDS (v3) to extract discovery data from the management server. Health
reporting (and setting endpoint state) is doing via the Health Disovery Service
(HealthCheckRequestOrEndpointHealthResponse), where we send EndpointHealthResponses to it.

The "admin" site of this tool (add, rm) isn't implemented yet, because I can't find the protobufs
that I need to implement this.

## See Also

<https://github.com/miekg/xdsd> is an xDS endpoint that implements a sane set of action upon
receiving these requests.

## Bugs

What if you drain a cluster and then a new healthy end point is added?
