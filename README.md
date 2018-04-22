[![Build
Status](https://travis-ci.org/google/cloudprober.svg?branch=master)](https://travis-ci.org/google/cloudprober)
[![Build status](https://ci.appveyor.com/api/projects/status/ypg1okxxfedwkksk?svg=true)](https://ci.appveyor.com/project/manugarg/cloudprober-wwcpu)

# Cloudprober

[cloudprober.org](https://cloudprober.org)

Cloudprober is a monitoring software that makes it super-easy to monitor
availability and performance of various components of your system. Cloudprober
employs the "active" monitoring model. It runs probes against (or on) your
components to verify that they are working as expected. For example, it can run
a probe to verify that your frontends can reach your backends. Similarly it can
run a probe to verify that your in-Cloud VMs can actually reach your on-premise
systems. This kind of monitoring makes it possible to monitor your systems'
interfaces regardless of the implementation and helps you quickly pin down
what's broken in your system.

![Cloudprober Use
Case](https://cloudprober.org/diagrams/cloudprober_use_case.svg)

## Features

*   Automated target discovery for Cloud targets. GCP is supported
    out-of-the-box; other Cloud providers can be added easily.
*   Integration with [StackDriver](https://cloud.google.com/stackdriver/). If
    configured, Cloudprober exports probe results to StackDriver as custom
    metrics.
*   Integration with open source monitoring stack of
    [Prometheus](http://prometheus.io) and [Grafana](http://grafana.com).
    Cloudprober exports probe results as counter based metrics that work well
    with Prometheus and Grafana.
*   Built-in implementations for the most common probe types: PING, HTTP, UDP,
    DNS.
*   Arbitrary, complex probes can be run through the external probe type. For
    example, you could write a simple script to insert and delete a row in your
    database, and execute this script through the 'EXTERNAL' probe type.
*   Fast and efficient ping prober implementation that allows probing thousands
    of hosts with minimal resources.
*   Strong focus on ease of deployment. Cloudprober is written entirely in Go
    and compiles into a static binary. It can be easily deployed through docker
    containers. Thanks to the automated target discovery, there is usually no
    need to re-deploy or re-configure cloudprober in response to the most of the
    changes.
*   Go templates based config file with substitutions for standard variables
    like project, zone, instance names etc allows for using same config file
    across the fleet.
*   Low footprint. Cloudprober docker image is small, containing just the
    statically compiled binary and it takes very little CPU and RAM to run even
    a large number of probes.
*   Extensible architecture. Cloudprober can be easily extended along most of
    the dimensions. Adding support for other Cloud targets, monitoring systems
    (e.g. Graphite, Amazon Cloudwatch) and even a new probe type, is
    straight-forward and fairly easy.

Visit Cloudprober's website at [cloudprober.org](https://cloudprober.org) to get
started with Cloudprober.

Join [Cloudprober Group](https://groups.google.com/forum/#!forum/cloudprober) for
discussion and release announcements.
