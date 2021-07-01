---
menu:
    main:
        parent: "How-Tos"
        weight: 26
title: "Additional Labels"
date: 2021-06-28T17:24:32-07:00
---
## Adding additional labels

You can add additional labels to probe metrics using a probe-level field: `additional_label`. An additional label's value can be static, or it can be determined at the run-time: from the environment that the probe is running in (e.g. GCE instance labels), or target's labels.

Example config [here](https://github.com/google/cloudprober/blob/master/examples/additional_label/cloudprober.cfg) demonstrates adding various types of additional labels to probe metrics. For this config (also listed below for quick rerefence):

* if ingress target has label "`fqdn:app.example.com`",
* and prober is running in the GCE zone `us-east1-c`,
* and prober's GCE instance has label `env:prod`.

Probe metrics will look like the following:
```
 total{probe="my_ingress",ptype="http",metrictype="prober",env="prod",src_zone="us-east1-c",host="app.example.com"}: 90
 success{probe="my_ingress",ptype="http",metrictype="prober",env="prod",src_zone="us-east1-c",host="app.example.com"}: 80
```



```bash
probe {
  name: "my_ingress"
  type: HTTP

  targets {
    rds_targets {
      resource_path: "k8s://ingresses"
      filter {
        key: "namespace"
        value: "default"
      }
    }
  }
  
  # Static label
  additional_label {
    key: "metrictype"
    value: "prober"
  }
  
  # Label is configured at the run-time, based on the prober instance label (GCE).
  additional_label {
    key: "env"
    value: "{{.label_env}}"
  }
  
  # Label is configured at the run-time, based on the prober environment (GCE).
  additional_label {
    key: "src_zone"
    value: "{{.zone}}"
  }
  
  # Label is configured based on the target's labels.
  additional_label {
    key: "host"
    value: "@target.label.fqdn@"
  }

  http_probe {}
}
```
(Listing source: [examples/additional_label/cloudprober.cfg](https://github.com/google/cloudprober/blob/master/examples/additional_label/cloudprober.cfg))

## Adding your own metrics
For external probes, Cloudprober also allows external programs to provide additional metrics.
See [External Probe](https://cloudprober.org/how-to/external-probe) for more details.

