# xdsctl

Communicate with xDS endpoint.

## Usage

~~~
xdsctl [OPTIONS] VERB [VERB] [ARGS]
~~~

## List

~~~
xdsctl list cluster [CLUSTER]
~~~

Shows:

~~~
CLUSTER        TYPE
cluster-v0-0   EDS
cluster-v0-1   EDS
cluster-v0-2   EDS
cluster-v0-3   EDS
~~~

~~~
xdsctl list endpoints [CLUSTER]
~~~

## Drain

xdsctl drain cluster CLUSTER [ENDPOINT]
xdsctl drain cluster CLUSTER [ENDPOINT [HEALTH]] - sets endpoint health status to DRAIN

xsdctl drain region [REGION]
xdsctl drain zone [ZONE]
xdsctl drain subzone [SUBZONE] specify cluster

## Set

endpoint is identified by address, cluser identified by name

xdsctl set cluster weight|type CLUSTER WEIGHT[TYPE]

xdsctl set cluster CLUSTER  load|weight|health -c cluster -e endpoint load|weight|health


# rm, remove, delete??

xdsctl rm cluster CLUSTER [ENDPOINT]


# Race condition

what if you drain a cluster and then a new healthy end point is added?
