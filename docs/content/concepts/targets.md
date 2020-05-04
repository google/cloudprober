---
menu:
    main:
        name: "Targets Discovery"
        weight: 2
title: "Targets Discovery"
date: 2019-10-25T17:24:32-07:00
---
Automatic and continuous discovery of the targets is one of the core features of
Cloudprober. This feature is specially critical for the dynamic environments that today's cloud based deployments make possible. For exmaple, in a kubernetes cluster number of pods and their IPs can change on the fly, either in response to replica count changes or node failures. Automated targets discovery makes sure that we don't have to reconfigure Cloudprober in response to such events.

## Overview
The main idea behind Cloudprober's targets discovery is to use an independent source of truth to figure out the targets we are supposed to monitor. This source of truth is usually the resource provider's API, for example, GCE API and Kubernetes API. Cloudprober periodically polls these APIs to get the latest list of resources.

Example targets configuration:

```bash
targets {
  rds_targets {
    # Monitor all endpoints for the service service-a
    resource_path: "k8s://endpoints/service-a"
  }
}
```



Some salient features of the cloudprober's targets discovery:

* Continuous discovery. We don't just discover targets in the beginning, but keep refreshing them at a regular interval.
* Protection against the upstream provider failures. If refreshing of the targets fails during one of the refresh cycles, we continue using the existing set of targets.
* Targets data includes name, labels, IP and ports (labels and ports are optional). These details allow configuring probes, for example, automatically setting port in the HTTP probe, and using target lables for probe results labeling.
* Well-defined protobuf based API (RDS -- more on it below) to add support for more target types.
* Currently supports GCE and Kubernetes resources. Adding more resource types should be straightforward. 

Cloudprober currently (__as of v0.10.7__) supports following types of dynamic targets:

Resource Provider       | Resource Types 
-------------------------------|---------
Kubernetes (k8s)         | pods, endpoints, services      
GCP (gcp)                      | gce_instances, pubsub_messages 


## Resource Discovery Service

To provide a consistent interface between targets' configuration and the actual implementation, Cloudprober defines and uses a protocol called RDS (Resource Discovery Service). Cloudprober's targets module includes a targets type "rds_targets", that talks to an RDS backend that is either part of the same process or available over gRPC.


<a href="/diagrams/rds_targets.png"><img style="float: center;" width=450px src="/diagrams/rds_targets.png"></a>

Here are the RDS targets configuration ([RDSTargets](https://github.com/google/cloudprober/blob/86a1d1fcd2f8505c45ff462d69458fd5b9964e5f/targets/proto/targets.proto#L12)) options:

```protobuf
message RDSTargets {
  // RDS server options, for example:
  // rds_server_options {
  //   server_address: "rds-server.xyz:9314"
  //   oauth_config: {
  //     ...
  //   }
  // }
  // Default is to use the local server if any.
  optional rds.ClientConf.ServerOptions rds_server_options = 1;

  // Resource path specifies the resources to return. Resources paths have the
  // following format:
  // <resource_provider>://<resource_type>/<additional_params>
  //
  // Examples:
  // For GCE instances in projectA: "gcp://gce_instances/<projectA>"
  // Kubernetes Pods : "k8s://pods"
  optional string resource_path = 2;

  // Filters to filter resources by. Example:
  // filter {
  //   key: "namespace"
  //   value: "mynamesspace"
  // }
  // filter {
  //   key: "labels.app"
  //   value: "web-service"
  // }
  repeated rds.Filter filter = 3;

  // IP config to specify the IP address to pick for a resource. IPConfig
  // is defined here:
  // https://github.com/google/cloudprober/blob/master/rds/proto/rds.proto
  optional rds.IPConfig ip_config = 4;
}
```

Most options are explained in the comments for quick references. Here is the further explanation of some of these options:

### rds_server_options
This [field](https://github.com/google/cloudprober/blob/86a1d1fcd2f8505c45ff462d69458fd5b9964e5f/rds/client/proto/config.proto#L19) specifies how to connect to the RDS server: server address and security options (OAuth and TLS). If left unspecified, it connects to the local server if any (started through `rds_server` option). Next up it looks for the `rds_server_options` in [global_targets_options](https://github.com/google/cloudprober/blob/86a1d1fcd2f8505c45ff462d69458fd5b9964e5f/targets/proto/targets.proto#L125). 

### resource_path
Resource path specifies the resources we are interested in. It consists of _resource provider_, _resource type_ and optional _relative path_: `<resource_provider>://<resource_type>/<optional_relative_path>`
  * `resource_provider`: Resource provider is a generic concept within the RDS protocol but usually maps to the cloud provider. Cloudprober RDS server currently implements the Kubernetes (k8s) and GCP (gcp) resource providers. We plan to add more resource providers in future.
  * `resource_type`: Available resource types depend on the providers, for example, for k8s provider supports the following resource types: _pods_, _endpoints_, and _services_.
  * `optional_relative_path`: For most resource types you can specify resource name in the resource path itself, e.g. `k8s://services/cloudprober`. Alternatively, you can use filters to filter by name, resource, etc.

### filter
Filters are key-value strings that can be used to filter resources by various fields. Filters depend on the resource types, but most resources support filtering by name and labels.

```
# Return resources that start with "web" and have label "service:service-a"
...
 filter {
   key: "name"
   value: "^web.*"
 }
 filter {
   key: "labels.service"
   value: "service-a"
 }
```

* Filters supported by kubernetes resources: [k8s filters](https://github.com/google/cloudprober/blob/e4a0321d38d75fb4655d85632b52039fa7279d1b/rds/kubernetes/kubernetes.go#L55).
* Filters supported by GCP:
  * [GCE Instances](https://github.com/google/cloudprober/blob/e4a0321d38d75fb4655d85632b52039fa7279d1b/rds/gcp/gce_instances.go#L44)
  * [Pub/Sub Messages](https://github.com/google/cloudprober/blob/e4a0321d38d75fb4655d85632b52039fa7279d1b/rds/gcp/pubsub.go#L34)

## Running RDS Server

RDS server can either be run as an independent process or it can be part of the main prober process. Former mode is useful for large deployments where you may want to reduce the API upcall traffic (for example, to GCP). For example, if you run 1000+ prober processes, it will be much more economical, from the provider API quota usage point of view, to have a centralized RDS service with much fewer (2-3) instances instead of having each prober process make its own API upcalls.

RDS server can be added to a cloudprober process using `rds_server` stanza. If you're running RDS server in an independent process, you'll have to enable gRPC server in that process so that other instances can access RDS server remotely. This is done using the field `grpc_port`.

Here is an example RDS server configuration:

```shell
rds_server {
  # GCP provider to discover GCP resources.
  provider {
    gcp_config {
      # Projects to discover resources in.
      project: "google.com:test-project-1"
      project: "google.com:test-project-2"

      # Discover GCE instances in us-central1.
      gce_instances {
        zone_filter: "name = us-central1-*"
        re_eval_sec: 60  # How often to refresh, default is 300s.
      }

      # GCE forwarding rules.
      forwarding_rules {}
    }
  }

  # Kubernetes targets are further discussed at:
  # https://cloudprober.org/how-to/run-on-kubernetes/#kubernetes-targets
  provider {
    kubernetes_config {
      endpoints {}
    }
  }
}

# Enable gRPC server for RDS. Only required for remote access to RDS server.
grpc_port: 9314
```

For the remote RDS server setup, you can also secure the communication using [TLS certificates](https://github.com/google/cloudprober/blob/master/config/proto/config.proto#L91).

