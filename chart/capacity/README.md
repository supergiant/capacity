# Capacity

Capacity asserts through hardware autoscaling that every pod should have a home—but that underutilized infrastructure should not be paid for (and, therefore, should not be running).

## Prerequisites

* Kubernetes 1.10+

## Installing the Chart

To install the chart with the release `name my-release`:

```
$ helm isntall --name my-release supergiant/capacity \
        ----set-file config.kubescaler.raw=$PATH_TO_KUBESCALER_CONFIG \
        ----set-file config.userdata=$PATH_TO_USERDATA
```
