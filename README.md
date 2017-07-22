[![Build
Status](https://travis-ci.org/google/cloudprober.svg?branch=master)](https://travis-ci.org/google/cloudprober)

# Cloudprober

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
Case](https://cloudprober.github.io/diagrams/cloudprober_use_case.svg)

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
*   Abritrary, complex probes can be run through the external probe type. For
    example, you could write a simple script to insert and delete a row in your
    database, and execute this script through the 'EXTERNAL' probe type.
*   Fast and efficient ping prober implementation that allows probing thousands
    of hosts with minimal resources.
*   Strong focus on ease of deployment. Cloudprober is written entirely in Go
    and compiles into a static binary. It can be easily deployed through docker
    containers. Thanks to the automated target discovery, there is usually no
    need to re-deploy or re-configure cloudprober in response to the most of the
    changes.
*   Low footprint. Cloudprober docker image is small, containing just the
    statically compiled binary and it takes very little CPU and RAM to run even
    a large number of probes.
*   Extensible architecture. Cloudprober can be easily extended along most of
    the dimensions. Adding support for other Cloud targets, monitoring systems
    and even a new probe type, is straight-forward and fairly easy.

## Getting Started

Getting started with Cloudprober is as easy as running the following command:

```
docker run --net host -v /tmp:/tmp cloudprober/cloudprober
# Note: --net host provides better network performance and makes port forwarding
# management easier.
```

This will start cloudprober with only the "sysvars" module (no probes). It will
write metrics to the stdout in cloudprober's line protocol format (To be
documented). It will also start a Prometheus exporter that can be accessed at:
http://localhost:9313

If docker is not an option, you can also download the pre-built binaries
directly from the Github's
[releases](http://github.com/google/cloudprober/releases) page.

Since sysvars variables are not very interesting themselves, lets add a simple
config that probes Google's homepage:

```shell
# Write config to a file in /tmp
cat > /tmp/cloudprober.cfg <<EOF
probe {
  name: "google_homepage"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }
  interval_msec: 5000  # 5s
  timeout_msec: 1000   # 1s
}
EOF
```

You can have cloudprober use this config file using one of the following
methods:

```shell
docker run --net host -v /tmp/cloudprober.cfg:/etc/cloudprober.cfg \
    -v /tmp:/tmp cloudprober/cloudprober

# Non-docker
./cloudprober --config_file /tmp/cloudprober.cfg
```

You'll see probe metrics at the URL: http://hostname:9313/metrics (prometheus
exporter page) and at the stdout. Prometheus exporter page will look like the
following: _todo: add_screenshot_ and stdout will look like:

```
cloudprober 1500590430132947313 1500590520 labels=ptype=http,probe=google-http,dst=www.google.com sent=17 rcvd=17 rtt=1808357 timeouts=0 resp-code=map:code,200:17
cloudprober 1500590430132947314 1500590530 labels=ptype=sysvars,probe=sysvars hostname="manugarg-workstation" uptime=100
cloudprober 1500590430132947315 1500590530 labels=ptype=http,probe=google-http,dst=www.google.com sent=19 rcvd=19 rtt=2116441 timeouts=0 resp-code=map:code,200:19
```

Since this is not very interesting by itself, let's run a prometheus instance to
scrape our metrics and generate pretty graphs.

### Running Prometheus

Download prometheus binary from the prometheus [release
page](https://prometheus.io/download/). You can use a config like the following
to scrape cloudprober running on the local host.

```shell
# Write config to a file in /tmp
cat > /tmp/prometheus.yml <<EOF
scrape_configs:
  # prometheus itself, useful for debugging
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
  - job_name: 'cloudprober'
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9313']
EOF

Start prometheus:
./prometheus --config.file=/tmp/prometheus.yml
```

Prometheus provides a web interface to interact with the data. You can access it
at http://localhost:9090. You can explore the probe metrics and build useful
graphs through this interface. All core probes in cloudprober export at least 3
counters:

*   _sent_: Number of requests sent (type of request depends on the probe type)
*   _rcvd_: Number of responses received.
*   _rtt_: Total round trip time in microseconds.

Using these counters, loss and latency can be calculated as:

```
loss    = (rate(sent) - rate(rcvd)) / rate(sent)
latency = rate(rtt) / rate(rcvd)
```

Assuming that prometheus is running at localhost:9090, graphs depicting loss
ratio and latency over time can be accessed in prometheus at: [loss and
latency](http://localhost:9090/graph?g0.range_input=1h&g0.expr=\(rate\(sent%5B1m%5D\)+-+rate\(rcvd%5B1m%5D\)\)+%2F+rate\(sent%5B1m%5D\)&g0.tab=0&g1.range_input=1h&g1.expr=rate\(rtt%5B1m%5D\)+%2F+rate\(rcvd%5B1m%5D\)+%2F+1000&g1.tab=0).
