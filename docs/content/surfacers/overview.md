---
menu:
    main:
        parent: "Exporting Metrics (Surfacers)"
        weight: 20
title: "Surfacers"
date: 2020-11-01T17:24:32-07:00
---
One of the biggest strengths of cloudprober is that it can export data to multiple monitoring systems, even simultaneously, just based on simple configuration. Cloudprober does that using a built-in mechanism, called surfacers. Each surfacer type implements interface for a specific monitoring system, for example, [_pubsub_](https://github.com/google/cloudprober/blob/master/surfacers/prometheus/proto/config.proto) surfacer publishes data to Google Pub/Sub. You can configure multiple surfacers at the same time. If you don't specify any surfacer, [_prometheus_](https://github.com/google/cloudprober/blob/master/surfacers/prometheus/proto/config.proto) and [_file_](https://github.com/google/cloudprober/blob/master/surfacers/file/proto/config.proto) surfacers are enabled automatically.

Why other monitoring systems? Cloudprober's main purpose is to run probes and build standard, usable metrics based on the results of those probes. It doesn't take any action on the generated data. Instead, it provides an easy interface to make that probe data available to systems that provide ways to consume monitoring data, for example for graphing and alerting.

Cloudprober currently supports following surfacer types:

* Prometheus ([config](https://github.com/google/cloudprober/blob/master/surfacers/prometheus/proto/config.proto))
* [Stackdriver (Google Cloud Monitoring)](/surfacers/stackdriver)
* Google Pub/Sub ([config](https://github.com/google/cloudprober/blob/master/surfacers/pubsub/proto/config.proto))
* Postgres ([config](https://github.com/google/cloudprober/blob/master/surfacers/postgres/proto/config.proto))
* File ([config](https://github.com/google/cloudprober/blob/master/surfacers/file/proto/config.proto))
* [Cloudwatch (AWS Cloud Monitoring)](/surfacers/cloudwatch)

Source: [surfacers config](https://github.com/google/cloudprober/blob/7bc30b62e42f3fe4e8a2fb8cd0e87ea18b73aeb8/surfacers/proto/config.proto#L14).

It's easy to add more surfacers without having to understand the internals of cloudprober. You only need to implement the [Surfacer interface](https://github.com/google/cloudprober/blob/7bc30b62e42f3fe4e8a2fb8cd0e87ea18b73aeb8/surfacers/surfacers.go#L87).

## Configuration

Adding surfacers to cloudprober is as easy as adding "surfacer" config stanzas to your config, like the following:

```shell
# Enable prometheus and stackdriver surfacers.

# Make probe metrics available at the URL :<cloudprober_port>/metrics, for
# scraping by prometheus.
surfacer {
  type: PROMETHEUS

  prometheus_surfacer {
    # Following option adds a prefix to exported metrics, for example,
    # "total" metric is exported as "cloudprober_total".
    metrics_prefix: "cloudprober"
  }
}

# Stackdriver (Google Cloud Monitoring) surfacer. No other configuration
# is necessary if running on GCP.
surfacer {
  type: STACKDRIVER
}
```

### Filtering Metrics

It is possible to filter the metrics that the surfacers receive.

#### Filtering by Label

Cloudprober can filter the metrics that are published to surfacers. To filter metrics by labels, reference one of the following keys in the surfacer configuration:

- `allow_metrics_with_label`
- `ignore_metrics_with_label`

_Note: `ignore_metrics_with_label` takes precedence over `allow_metrics_with_label`._

For example, to ignore all sysvar metrics:

```
surfacer {
  type: PROMETHEUS

  ignore_metrics_with_label {
    key: "probe",
    value: "sysvars",
  }
}
```
Or to only allow metrics from http probes:

```
surfacer {
  type: PROMETHEUS

  allow_metrics_with_label {
    key: "ptype",
    value: "http",
  }
}
```

#### Filtering by Metric Name

For certain surfacers, cloudprober can filter the metrics that are published by name. The surfacers that support this functionality are:

- Cloudwatch
- Prometheus
- Stackdriver

Within the surfacer configuration, the following options are defined:

- `allow_metrics_with_name`
- `ignore_metrics_with_name`

_Note: `ignore_metrics_with_name` takes precedence over `allow_metrics_with_name`._

To filter out all `validation_failure` metrics by name:

```
surfacer {
  type: PROMETHEUS

  ignore_metrics_with_name: "validation_failure"
}
```
(Source: https://github.com/google/cloudprober/blob/master/surfacers/proto/config.proto)
