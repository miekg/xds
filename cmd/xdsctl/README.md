# xdsctl

## Name

xdsctl - communicate with a xDS endpoint.

The are several commands implemented, just look at the help output of xdsctl (which should be fairly
complete).

The "admin" site of this tool (add, rm) isn't implemented yet, because I can't find the protobufs
that I need to implement this; this can be worked around by using CDS and or EDS to send a
"discovery response" that's seen as a cue to add or remove to a different endpoint.
