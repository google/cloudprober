---
menu:
    main:
        parent: "How-Tos"
        weight: 25
title: "External Probe"
date: 2017-10-08T17:24:32-07:00
---
External probe type allows you to run arbitrary, complex probes through
Cloudprober. An external probe runs an independent external program for actual
probing. Cloudprober calculates probe metrics based on program's exit status
and time elapsed in execution. 

Cloudprober also allows external programs to provide additional metrics.
Every message send to `stdout` will be parsed as a new metrics to be emitted.
Thus for your general logging you can use another I/O stream  like `stderr`.

## Sample Probe
To understand how it works, lets create a sample probe that sets and gets a key
in a redis server. Here is the `main` function of such a probe:

{{< highlight go >}}
func main() {
    var client redis.Client
    var key = "hello"
    startTime := time.Now()
    client.Set(key, []byte("world"))
    fmt.Printf("set_latency_ms %f\n", float64(time.Since(startTime).Nanoseconds())/1e6)

    startTime = time.Now()
    val, _ := client.Get("hello")
    log.Printf("%s=%s", key, string(val))
    fmt.Printf("get_latency_ms %f\n", float64(time.Since(startTime).Nanoseconds())/1e6)
}
{{< / highlight >}}

(Full listing: https://github.com/google/cloudprober/blob/master/examples/external/redis_probe.go)

This program sets and gets a key in redis and prints the time taken for both operations. Cloudprober can use this program as an external probe, to verify
the availability and performance of the redis server. This program assumes that
redis server is running locally, at its default port. For the sake of demonstration, lets run a local redis server (you can also easily modify this program to use a different server.)

{{< highlight bash >}}
#!bash
brew install redis
{{< / highlight >}}

Let's compile our probe program (redis_probe.go) and verify that it's working
as expected:

{{< highlight bash >}}
#!bash
CGO_ENABLED=0 go build -ldflags “-extldflags=-static” ./redis_probe.go
./redis_probe
set_latency_ms 22.656588
2018/02/26 15:16:14 hello=world
get_latency_ms 2.173560
{{< / highlight >}}


## Configuration
Here is the external probe configuration that makes use of this program:

Full example in [examples/external/cloudprober.cfg](https://github.com/google/cloudprober/blob/master/examples/external/cloudprober.cfg).

{{< highlight shell >}}
# Run an external probe that executes a command from the current working
# directory.
probe {
  name: "redis_probe"
  type: EXTERNAL
  targets { dummy_targets {} }
  external_probe {
    mode: ONCE
    command: "./redis_probe"
  }
}
{{< / highlight >}}

Note: To pass target information to your external program, you can send target information as arguments using the `@label@` notation.  
Supported field are: target, address, port, probe, and your target.labels like target.label.fqdn.
```
command: "./redis_probe" -host=@address@ -port=@port@
```

Running it through cloudprober, you'll see the following output:

{{< highlight bash >}}
# Launch cloudprober
cloudprober --config_file=cloudprober.cfg

cloudprober 1519..0 1519583408 labels=ptype=external,probe=redis_probe,dst= success=1 total=1 latency=12143.765
cloudprober 1519..1 1519583408 labels=ptype=external,probe=redis_probe,dst= set_latency_ms=0.516 get_latency_ms=0.491
cloudprober 1519..2 1519583410 labels=ptype=external,probe=redis_probe,dst= success=2 total=2 latency=30585.915
cloudprober 1519..3 1519583410 labels=ptype=external,probe=redis_probe,dst= set_latency_ms=0.636 get_latency_ms=0.994
cloudprober 1519..4 1519583412 labels=ptype=external,probe=redis_probe,dst= success=3 total=3 latency=42621.871
{{< / highlight >}}

You can import this data in prometheus following the process outlined at:
[Running Prometheus]({{< ref "/getting-started.md#running-prometheus" >}}). Before doing that, let's make it more interesting.

## Distributions
How nice will it be if we could find distribution of the set and get latency. If tail latency was too high, it could explain the random timeouts in your application. Fortunately, it's very easy to create distributions in Cloudprober. You just need to add the following section to your probe definition:

Full example in [examples/external/cloudprober_aggregate.cfg](https://github.com/google/cloudprober/blob/master/examples/external/cloudprober_aggregate.cfg).

{{< highlight shell >}}
# Run an external probe and aggregate metrics in cloudprober.
...
output_metrics_options {
  aggregate_in_cloudprober: true

  # Create distributions for get_latency_ms and set_latency_ms.
  dist_metric {
    key: "get_latency_ms"
    value: {
      explicit_buckets: "0.1,0.2,0.4,0.6,0.8,1.0,2.0"
    }
  }
  dist_metric {
    key: "set_latency_ms"
    value: {
      explicit_buckets: "0.1,0.2,0.4,0.6,0.8,1.0,2.0"
    }
  }
}
{{< / highlight >}}

This configuration adds options to aggregate the metrics in the cloudprober and configures "get\_latency\_ms" and "set\_latency\_ms" as distribution metrics with explicit buckets. Cloudprober will now build cumulative distributions using
for these metrics. We can import this data in Stackdriver or Prometheus and get the percentiles of the "get" and "set" latencies. Following screenshot shows the
grafana dashboard built using these metrics.

<a href="/diagrams/redis_probe_screenshot.png"><img style="float: center;" width=300px src="/diagrams/redis_probe_screenshot.png"></a>

## Server Mode

The probe that we created above forks out a new `redis_probe` process for every
probe cycle. This can get expensive if probe frequency is high and the process is big (e.g. a Java binary). Also, what if you want to keep some state across probes, for example, lets say you want to monitor performance over HTTP/2 where you keep using the same TCP connection for multiple HTTP requests. A new process
every time makes keeping state impossible.

External probe's server mode provides a way to run the external probe process in daemon mode. Cloudprober communicates with this process over stdout/stdin (connected with OS pipes), using serialized protobuf messages. Cloudprober comes with a serverutils package that makes it easy to build external probe servers in Go.

![External Probe Server](/diagrams/external_probe_server.svg)

Please see the code at 
[examples/external/redis_probe.go](https://github.com/google/cloudprober/blob/master/examples/external/redis_probe.go) for server mode implementation of the above probe. Here is the corresponding
cloudprober config to run this probe in server mode: [examples/external/cloudprober_server.cfg](
https://github.com/google/cloudprober/blob/master/examples/external/cloudprober_server.cfg).

In server mode, if external probe process dies for reason, it's restarted by Cloudprober.
