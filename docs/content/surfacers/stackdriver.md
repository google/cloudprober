---
menu:
    main:
        parent: "Exporting Metrics (Surfacers)"
        weight: 30
title: "Stackdriver (Google Cloud Monitoring)"
date: 2020-11-01T17:24:32-07:00
---
Cloudprober can natively export metrics to Google Cloud Monitoring (formerly, Stackdriver) using stackdriver [surfacer](/surfacers/overview).  Adding stackdriver surfacer to cloudprober is as simple as adding the following stanza to the config:

```
surfacer {
  type: STACKDRIVER
}
```

This config will work if you're running on GCP and your VM (or GKE pod) has access to Cloud Monitoring (Stackdriver). If running on any other platform, you'll  have to specify the GCP project where you want to send the metrics, and you'll have to configure your environment for [Google Application Default Credentials](https://cloud.google.com/docs/authentication/production#automatically).

By default, stackdriver surfacer exports metrics with the following prefix: `custom.googleapis.com/cloudprober/<probe-type>/<probe>`. For example, for HTTP probe named `google_com`, standard metrics will be exported as:

 ```
custom.googleapis.com/cloudprober/http/google_com/total
custom.googleapis.com/cloudprober/http/google_com/success
custom.googleapis.com/cloudprober/http/google_com/failure
custom.googleapis.com/cloudprober/http/google_com/latency
 ```

Here are all the config options for stackdriver surfacer:

```protobuf
  // GCP project name for stackdriver. If not specified and running on GCP,
  // local project is used.
  optional string project = 1;

  // If allowed_metrics_regex is specified, only metrics matching the given
  // regular expression will be exported to stackdriver. Since probe type and
  // probe name are part of the metric name, you can use this field to restrict
  // stackdriver metrics to a particular probe.
  // Example:
  // allowed_metrics_regex: ".*(http|ping).*(success|validation_failure).*"
  optional string allowed_metrics_regex = 3;

  // Monitoring URL base. Full metric URL looks like the following:
  // <monitoring_url>/<ptype>/<probe>/<metric>
  // Example:
  // custom.googleapis.com/cloudprober/http/google-homepage/latency
  optional string monitoring_url = 4
      [default = "custom.googleapis.com/cloudprober/"];
```

(Source: https://github.com/google/cloudprober/blob/master/surfacers/stackdriver/proto/config.proto)

For example, you can configure stackdriver surfacer to export only metrics that match a specific regex: 

```protobuf
surfacer {
  stackdriver_surfacer {
    # Export only "http" probe metrics.
    allowed_metrics_regex: ".*\\/http\\/.*"
  }
}
```



## Accessing the data

Cloudprober exports metrics to stackdriver as [custom metrics](https://cloud.google.com/monitoring/custom-metrics). Since all cloudprober metrics are counters (total number of probes, success, latency), you'll see rates of these metrics in stackdriver [metrics explorer](https://cloud.google.com/monitoring/charts/metrics-explorer) by default. This data may not be very useful as it is (unless you're using distributions in cludprober, more on that later).

However, stackdriver now provides a powerful monitoring query language,[MQL](https://cloud.google.com/monitoring/mql), using which we can get more useful metrics.

MQL to get failure ratio:

```shell
fetch global
| { metric 'custom.googleapis.com/cloudprober/http/google_com/failure'
  ; metric 'custom.googleapis.com/cloudprober/http/google_com/total' }
| align delta(1m)
| join
| div
```

MQL to get average latency for a probe:

```shell
fetch global
| { metric 'custom.googleapis.com/cloudprober/http/google_com/latency'
  ; metric 'custom.googleapis.com/cloudprober/http/google_com/success' }
| align delta(1m)
| join
| div
```

You can use MQL to create graphs and generate alerts. Note that in the examples here we are fetching from the "global" source (_fetch global_); if you're running on GCP, you can improve performance of your queries by specifying the "gce_instance" resource type: _fetch gce_instance_.

