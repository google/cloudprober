---
menu:
    main:
        weight: -90
title: Getting Started
---

## Installation

* __From Source__  
If you have Go 1.9 or higher installed and GOPATH environment variable properly set up, you
can download and install `cloudprober` using the following commands:

  ```bash
  go get github.com/google/cloudprober
  GOBIN=$GOPATH/bin go install $GOPATH/src/github.com/google/cloudprober/cmd/cloudprober.go
  ```

* __Pre-built Binaries__  
You can download pre-built binaries for Linux, MacOS and Windows from the
project's [releases page](http://github.com/google/cloudprober/releases).  
(_See [this page](
https://github.com/google/cloudprober/wiki/Download-unreleased-binaries) on
how to download unreleased binaries._)

* __Docker Image__
You can download and run the [latest](
https://github.com/google/cloudprober/wiki/Docker-versions) docker image using
the following command:

  ```bash
  docker run --net host cloudprober/cloudprober
  # Note: "--net host" provides better network performance and makes port
  # forwarding management easier, but is not required.
  ```

## Configuration

Without any config, cloudprober will run only the "sysvars" module (no probes) and write metrics to stdout in cloudprober's line protocol format (to be documented). It will also start a [Prometheus](http://prometheus.io) exporter at: http://localhost:9313 (you can change the default port through the environment variable *CLOUDPROBER_PORT* and the default listening address through the environment variable *CLOUDPROBER_HOST*).

Since sysvars variables are not very interesting themselves, lets add a simple
config that probes Google's homepage:

```bash
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

This config adds an HTTP probe that accesses the homepage of the target "www.google.com" every 5s with a timeout of 1s. Cloudprober configuration is specified in the text protobuf format, with config schema described by the proto file: [config.proto](https://github.com/google/cloudprober/blob/master/config/proto/config.proto).

Assuming that you saved this file at `/tmp/cloudprober.cfg` (following the command above), you can have cloudprober use this config file using the following command line:

```bash
./cloudprober --config_file /tmp/cloudprober.cfg
```

You can have the standard docker image use this config using the following
command:

```bash
docker run --net host -v /tmp/cloudprober.cfg:/etc/cloudprober.cfg \
    cloudprober/cloudprober
```

Note: While running on GCE, cloudprober config can also be provided through a
custom metadata attribute: __cloudprober\_config__.

## Verification

One quick way to verify that cloudprober got the correct config is to access the URL `http://localhost:9313/config` (through cURL or in browser). It returns the config that cloudprober is using. You can also look at its current status at the URL (_replace localhost by the actual hostname if not running locally_): `http://localhost:9313/status`.

You should be able to see the generated metrics at `http://localhost:9313/metrics` (prometheus format) and the stdout (cloudprober format):

```bash
cloudprober 15.. 1500590520 labels=ptype=http,probe=google-http,dst=.. total=17 success=17 latency=180835
cloudprober 15.. 1500590530 labels=ptype=sysvars,probe=sysvars hostname="manugarg-ws" uptime=100
cloudprober 15.. 1500590530 labels=ptype=http,probe=google-http,dst=.. total=19 success=19 latency=211644
```

This information is good for debugging monitoring issues, but to really make sense of this data, you'll need to feed this data to another monitoring system like _Prometheus_ or _StackDriver_ (see [Surfacers](/surfacers/overview) for more details). Lets set up a Prometheus and Grafana stack to make pretty graphs for us.

## Running Prometheus

Download prometheus binary from its [release page](https://prometheus.io/download/). You can use a config like the following
to scrape a cloudprober instance running on the same host.

```bash
# Write config to a file in /tmp
cat > /tmp/prometheus.yml <<EOF
scrape_configs:
  - job_name: 'cloudprober'
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9313']
EOF

# Start prometheus:
./prometheus --config.file=/tmp/prometheus.yml
```

Prometheus provides a web interface at http://localhost:9090. You can explore probe metrics and
build useful graphs through this interface. All probes in cloudprober export at least 3 counters:

* _total_: Total number of probes.
* _success_: Number of successful probes. Difference between _total_ and
_success_ indicates failures.
* _latency_: Total (cumulative) probe latency.

Using these counters, probe failure ratio and average latency can be calculated
as:

```bash
failure_ratio = (rate(total) - rate(success)) / rate(total)
avg_latency   = rate(latency) / rate(success)
```

Assuming that prometheus is running at `localhost:9090`, graphs depicting
failure ratio and latency over time can be accessed in prometheus at: [this url ]("http://localhost:9090/graph?g0.range_input=1h&g0.expr=(rate(total%5B1m%5D)+-+rate(success%5B1m%5D))+%2F+rate(total%5B1m%5D)&g0.tab=0&g1.range_input=1h&g1.expr=rate(latency%5B1m%5D)+%2F+rate(success%5B1m%5D)+%2F+1000&g1.tab=0").
Even though prometheus provides a graphing interface, Grafana provides much
richer interface and has excellent support for prometheus.

## Grafana

[Grafana](https://grafana.com) is a popular tool for building monitoring dashboards. Grafana has native support for prometheus and thanks to the excellent support for prometheus in Cloudprober itself, it's a breeze to build Grafana dashboards from Cloudprober's probe results.

To get started with Grafana, follow the Grafana-Prometheus
[integration guide](https://prometheus.io/docs/visualization/grafana/).
