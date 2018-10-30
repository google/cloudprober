---
menu:
    main:
        parent: "How-Tos"
        weight: 35
title: "Extending Cloudprober"
date: 2018-10-29T17:24:32-07:00
---
Cloudprober allows you to extend it across "probe" and "target" dimensions,
that is, you can add new probe and target types to it without having to fork
the entire codebase. Note that to extend cloudprober in this way, you will have
to maintain your own cloudprober binary (which is mostly a wrapper around the
"cloudprober package"), but you'll be able to use rest of the cloudprober code
from the common location.

## Sample probe type

To demonstrate how it works, let's add a new probe-type to Cloudprober. We'll
take the sample redis probe that we added in the
[external probe how-to]({{< ref "/how-to/external-probe.md#sample-probe" >}}),
and convert it into a probe type that one can easily re-use. Let's say that
this probe-type provides a way to test redis server functionality and it takes
the following options - operation (GET vs SET vs DELETE), key, value. This
probe's configuration looks like this:

{{< highlight bash >}}
probe {
  name: "redis_set"
  type: EXTENSION
  targets {
    host_names: "localhost:6379"
  }
  redis_probe {
    op: "set"
    key: "testkey"
    value: "testval"
  }
}
{{< / highlight >}}

To make cloudprober understand this config, we'll have to do a few things:

* Define the probe config in a protobuf (.proto) file and mark it as an
  extension of the overall config.

* Implement the probe type, possibly as a Go package, even though it can
  be embedded directly into the top-level binary.

* Create a new cloudprober binary that includes the new probe type package.

## Protobuf for the new probe type

Let's create a new directory for our code: `$GOPATH/src/myprober`.

{{< highlight protobuf >}}
// File: $GOPATH/src/myprober/myprobe/myprobe.proto

syntax = "proto2";

import "github.com/google/cloudprober/probes/proto/config.proto";

package myprober;

message ProbeConf {
  // Redis operation
  required string op = 1;

  // Key and value for the redis operation
  required string key = 2;
  optional string value = 3;
}

extend cloudprober.probes.ProbeDef {
  optional ProbeConf redis_probe = 200;
}

{{< / highlight >}}

Let's generate Go code for this protobuf:

{{< highlight shell >}}
# From the myprober directory
protoc --go_out=.,import_path=myprobe:. --proto_path=$GOPATH/src:. myprobe/*.proto

$ ls myprobe/
myprobe.pb.go  myprobe.proto
{{< /highlight >}}

## Implement the probe type

Now let's implement our probe type. Our probe type should implement the
[probes.Probe](https://godoc.org/github.com/google/cloudprober/probes#Probe) interface.

{{< highlight go >}}
// Full listing:
// github.com/google/cloudprober/examples/extensions/myprober/myprobe/myprobe.go

package myprobe

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
  name    string
  c       *configpb.ProbeConf
  targets []string
  opts    *options.Options
  ...
}

// Init initializes the probe with the given params.
func Init(name string, opts *options.Options) error {
  c, ok := opts.ProbeConf.(*ProbeConf)
  if !ok {
    return fmt.Errorf("not a my probe config")
  }
  // initialize p fields, p.name = name, etc.
}

// Start runs the probe indefinitely, at the configured interval.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
  probeTicker := time.NewTicker(p.opts.Interval)

  for {
    select {
    case <-ctx.Done():
      probeTicker.Stop()
      return
    case <-probeTicker.C:
      for _, em := range p.res {
        dataChan <- em
      }
      p.targets = p.opts.Targets.List()
      ...
      probeCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(p.opts.Timeout))
      p.runProbe(probeCtx)
      cancelFunc()
    }
  }
}

// runProbe runs probe for all targets and update EventMetrics.
func (p *Probe) runProbe(ctx context.Context) {
  p.targets = p.opts.Targets.List()

  var wg sync.WaitGroup
  for _, target := range p.targets {
    wg.Add(1)

    go func(target string, em *metrics.EventMetrics) {
      defer wg.Done()
      em.Metric("total").AddInt64(1)
      start := time.Now()
      err := p.runProbeForTarget(ctx, target) // run probe just for a single target
      if err != nil {
        p.l.Errorf(err.Error())
        return
      }
      em.Metric("success").AddInt64(1)
      em.Metric("latency").AddFloat64(time.Now().Sub(start).Seconds() / p.opts.LatencyUnit.Seconds())
    }(target, p.res[target])

  }

  wg.Wait()
}
{{< / highlight >}}

(Full listing: <https://github.com/google/cloudprober/blob/master/examples/extensions/myprober/myprobe/myprobe.go>)

This probe type sets or gets (depending on the configuration) a key-valye in
redis and records success and time taken (latency) if operation is successful.

## Implement a cloudprober binary that includes support for our probe

{{< highlight go >}}
// Full listing:
// github.com/google/cloudprober/examples/extensions/myprober/myprober.go

package main

...

func main() {
  flag.Parse()

  // Register our probe type
  probes.RegisterProbeType(int(myprobe.E_RedisProbe.Field),
                           func() probes.Probe { return &myprobe.Probe{} })

  err := cloudprober.InitFromConfig(getConfig()) // getConfig not shown here.
  if err != nil {
    glog.Exitf("Error initializing cloudprober. Err: %v", err)
  }

  // web.Init sets up web UI for cloudprober.
  web.Init()

  cloudprober.Start(context.Background())

  // Wait forever
  select {}
}
{{< / highlight >}}
(Full listing: <https://github.com/google/cloudprober/blob/master/examples/extensions/myprober/myprober.go>)

Let's write a test config that uses the newly defined probe type:

{{< highlight bash >}}
probe {
  name: "redis_set"
  type: EXTENSION
  interval_msec: 10000
  timeout_msec: 5000
  targets {
    host_names: "localhost:6379"
  }
  [myprober.redis_probe] {
    op: "set"
    key: "testkey"
    value: "testval"
  }
}
{{< / highlight >}}
(Full listing: <https://github.com/google/cloudprober/blob/master/examples/extensions/myprober/myprober.cfg>)

Let's compile our prober and run it with the above config:

{{< highlight bash >}}
go build ./myprober.go
./myprober --config_file=myprober.cfg
{{< / highlight >}}

you should see an output like the following:
```
cloudprober 1540848577649139842 1540848587 labels=ptype=redis,probe=redis_set,dst=localhost:6379 total=31 success=31 latency=70579.823
cloudprober 1540848577649139843 1540848887 labels=ptype=sysvars,probe=sysvars hostname="manugarg-macbookpro5.roam.corp.google.com" start_timestamp="1540848577"
cloudprober 1540848577649139844 1540848887 labels=ptype=sysvars,probe=sysvars uptime_msec=310007.784 gc_time_msec=0.000 mallocs=14504 frees=826
cloudprober 1540848577649139845 1540848887 labels=ptype=sysvars,probe=sysvars goroutines=12 mem_stats_sys_bytes=7211256
cloudprober 1540848577649139846 1540848587 labels=ptype=redis,probe=redis_set,dst=localhost:6379 total=32 success=32 latency=72587.981
cloudprober 1540848577649139847 1540848897 labels=ptype=sysvars,probe=sysvars hostname="manugarg-macbookpro5.roam.corp.google.com" start_timestamp="1540848577"
cloudprober 1540848577649139848 1540848897 labels=ptype=sysvars,probe=sysvars uptime_msec=320006.541 gc_time_msec=0.000 mallocs=14731 frees=844
cloudprober 1540848577649139849 1540848897 labels=ptype=sysvars,probe=sysvars goroutines=12 mem_stats_sys_bytes=7211256
```

You can import this data in prometheus following the process outlined at:
[Running Prometheus]({{< ref "getting-started.md#running-prometheus" >}}).

## Conclusion

The article shows how to add a new probe type to cloudprober. Extending
cloudprober allows you to implement new probe types that may make sense for your
organization, but not for the open source community. You have to implement the
logic for the probe type, but other cloudprober features work as it is --
targets, metrics (e.g. latency distribution if you configure it), surfacers -
data can be multiple systems simultaneously, etc.


