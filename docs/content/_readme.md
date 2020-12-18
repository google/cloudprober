[![Build
Status](https://travis-ci.org/google/cloudprober.svg?branch=master)](https://travis-ci.org/google/cloudprober)
[![Build status](https://ci.appveyor.com/api/projects/status/ypg1okxxfedwkksk?svg=true)](https://ci.appveyor.com/project/manugarg/cloudprober-wwcpu)
[![Docker Pulls](https://img.shields.io/docker/pulls/cloudprober/cloudprober.svg)](https://hub.docker.com/v2/repositories/cloudprober/cloudprober/)

Cloudprober is a monitoring software that makes it super-easy to monitor
availability and performance of various components of your system. Cloudprober
employs the "active" monitoring model. It runs probes against (or on) your
components to verify that they are working as expected. For example, it can run
a probe to verify that your frontends can reach your backends. Similarly it can
run a probe to verify that your in-Cloud VMs can actually reach your on-premise
systems. This kind of monitoring makes it possible to monitor your systems'
interfaces regardless of the implementation and helps you quickly pin down
what's broken in your system.

![Cloudprober Use Case](http://cloudprober.org/diagrams/cloudprober_use_case.svg)

## Features

*   Automated target discovery for Cloud targets. __GCE__ and __Kubernetes__
    are supported out-of-the-box, other Cloud providers can be added easily.
*   Integration with open source monitoring stack of
    [Prometheus](http://prometheus.io) and [Grafana](http://grafana.com).
    Cloudprober exports probe results as counter based metrics that work well
    with Prometheus and Grafana.
*   Integration with [StackDriver](https://cloud.google.com/stackdriver/). If
    configured, Cloudprober exports probe results to StackDriver as custom
    metrics.
*   Fast and efficient built-in implementations for the most common types of
    checks: PING (ICMP), HTTP, UDP, DNS. Especially PING and UDP probes are
    implemented in such a way that thousands of hosts can be probed with
    minimal resources.
*   Arbitrary, complex probes can be run through the external probe type. For
    example, you could write a simple script to insert and delete a row in your
    database, and execute this script through the 'EXTERNAL' probe type.
*   Standard metrics - _total_, _success_, _latency_. Latency can be configured
    to be a distribution (histogram) metric, allowing calculations of
    percentiles.
*   Strong focus on ease of deployment. Cloudprober is written entirely in Go
    and compiles into a static binary. It can be easily deployed, either as a
    standalone binary or through docker containers. Thanks to the automated,
    continuous, target discovery, there is usually no need to re-deploy or
    re-configure cloudprober in response to the most of the changes.
*   Low footprint. Cloudprober docker image is small, containing just the
    statically compiled binary and it takes very little CPU and RAM to run even
    a large number of probes.
*   Extensible architecture. Cloudprober can be easily extended along most of
    the dimensions. Adding support for other Cloud targets, monitoring systems
    and even a new probe type, is straight-forward and fairly easy.

## Getting Started

Visit [Getting Started](http://cloudprober.org/getting-started) page to get
started with Cloudprober.
    
We'd love to hear your feedback. If you're using Cloudprober, would you please
mind sharing how you use it by adding a comment [here](
https://github.com/google/cloudprober/issues/123). It will be a great help in
planning Cloudprober's future progression.

Join [Cloudprober Slack](https://join.slack.com/t/cloudprober/shared_invite/enQtNjA1OTkyOTk3ODc3LWQzZDM2ZWUyNTI0M2E4NmM4NTIyMjM5M2E0MDdjMmU1NGQ3NWNiMjU4NTViMWMyMjg0M2QwMDhkZGZjZmFlNGE) or [mailing list](
https://groups.google.com/forum/#!forum/cloudprober) for questions and discussion
about Cloudprober.
