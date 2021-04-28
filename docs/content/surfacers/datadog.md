---
menu:
    main:
        parent: "Exporting Metrics (Surfacers)"
        weight: 30
title: "Datadog (Metrics)"
date: 2021-04-28T17:00:00-07:00
---
Cloudprober can natively export metrics to Google Cloud Monitoring (formerly, Stackdriver) using stackdriver [surfacer](/surfacers/overview).  Adding stackdriver surfacer to cloudprober is as simple as adding the following stanza to the config:

Cloudprober can natively export metrics to Datadog using the Datadog [submit-metrics](https://docs.datadoghq.com/api/latest/metrics/#submit-metrics) API. Adding Datadog surfacer to cloudprober requires an existing account with Datadog, an [API\_KEY, and an APP\_KEY](https://docs.datadoghq.com/account_management/api-app-keys/). 

```
surfacer {
  type: DATADOG

  app_key: "abc123..."
  api_key: "def456..."
  prefix: "prefix.for.metrics"
}
```

By default, stackdriver surfacer exports metrics with the following prefix: `custom.googleapis.com/cloudprober/<probe-type>/<probe>`. For example, for HTTP probe named `google_com`, standard metrics will be exported as:

Datadog metrics will be exported as: 

 ```
prefix.for.metrics.total
prefix.for.metrics.success
prefix.for.metrics.latency
 ```

## Accessing the data

Cloudprober exports metrics to Datadog as [metrics](https://docs.datadoghq.com/metrics/). All metrics sent to datadog are sent as gauged. When cloudprober generates Distribution metrics to send to datadog, these metrics are sent as prefix.for.metrics.\<metric-name\>, one point for each point in the distribution with value equal to the lower bound of each bucket. Additionally, .sum and .count metrics are also created in Datadog. 

You can use the Datadog [metrics explorer](https://app.datadoghq.com/metric/explorer) to view the data or add the metrics to datadog dashboards.

