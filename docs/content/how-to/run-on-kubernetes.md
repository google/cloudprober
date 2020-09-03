---
menu:
    main:
        parent: "How-Tos"
        weight: 20
title: "Running On Kubernetes"
date: 2019-10-08T17:24:32-07:00
---

Kubernetes is a popular platform for running containers, and Cloudprober container runs on Kubernetes right out of the box. This document shows how you can use config map to provide config to cloudprober and reload cloudprober on config changes.

## ConfigMap

In Kubernetes, a convenient way to provide config to containers is to use config maps.  Let's create a config that specifies a probe to monitor "google.com".

```bash
probe {
  name: "google-http"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }
  http_probe {}
  interval_msec: 15000
  timeout_msec: 1000
}
```

Save this config in `cloudprober.cfg`, create a config map using the following command:

```bash
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg
```



## Deployment Map

Now let's add a `deployment.yaml` to add the config volume and cloudprober container:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudprober
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloudprober
  template:
    metadata:
      annotations:
        checksum/config: "${CONFIG_CHECKSUM}"
      labels:
        app: cloudprober
    spec:
      volumes:
      - name: cloudprober-config
        configMap:
          name: cloudprober-config
      containers:
      - name: cloudprober
        image: cloudprober/cloudprober
        command: ["/cloudprober"]
        args: [
          "--config_file","/cfg/cloudprober.cfg",
          "--logtostderr"
        ]
        volumeMounts:
        - name: cloudprober-config
          mountPath: /cfg
        ports:
        - name: http
          containerPort: 9313
---
apiVersion: v1
kind: Service
metadata:
  name: cloudprober
  labels:
    app: cloudprober
spec:
  ports:
  - port: 9313
    protocol: TCP
    targetPort: 9313
  selector:
    app: cloudprober
  type: NodePort
```

Note that we added an annotation to the deployment spec; this annotation allows us to update the deployment whenever cloudprober config changes. We can update this annotation based on the local cloudprober config content, and update the deployment using the following one-liner:

```bash
# Update the config checksum annotation in deployment.yaml before running
# kubectl apply.
CONFIG_CHECKSUM=$(kubectl get cm/cloudprober-config -o yaml | sha256sum) && \
cat deployment.yaml | envsubst | kubectl apply -f -
```

(Note: If you use Helm for Kubernetes deployments, Helm provides [a more native way](https://helm.sh/docs/howto/charts_tips_and_tricks/#automatically-roll-deployments) to include config checksums in deployments.)

Applying the above yaml file, should create a deployment with a service at port 9313:

```bash
$ kubectl get deployment
NAME          READY   UP-TO-DATE   AVAILABLE   AGE
cloudprober   1/1     1            1           94m

$ kubectl get service cloudprober
NAME          TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
cloudprober   NodePort   10.31.249.108   <none>        9313:31367/TCP   94m
```

Now you should be able to access various cloudprober URLs (`/status` for status,`/config` for config, `/metrics` for prometheus-format metrics) from within the cluster. For quick verification you can also set up a port forwarder and access these URLs locally at `localhost:9313`:

```
kubectl port-forward svc/cloudprober 9313:9313
```

Once you've verified that everything is working as expected, you can go on setting up metrics collection through prometheus (or stackdriver) in usual ways.



## Kubernetes Targets

If you're running on Kuberenetes, you'd probably want to monitor Kubernetes resources (e.g. pods, endpoints, etc) as well. Good news is that cloudprober supports dynamic [targets discovery](/concepts/targets/) of Kubernetes resources.

For example, following config adds HTTP probing of Kubernetes endpoints named 'cloudprober' (equivalent to _kubectl get ep cloudprober_).

```bash
probe {
  name: "pod-to-endpoints"
  type: HTTP

  targets {
    # RDS (resource discovery service) targets
    # Equivalent to kubectl get ep cloudprober
    rds_targets {
      resource_path: "k8s://endpoints/cloudprober"
    }
  }
  
  http_probe {
    resolve_first: true
    relative_url: "/status"
  }
}

# Run an RDS gRPC server to discover Kubernetes targets.
rds_server {
  provider {
    # For all options, please take a look at:
    # https://github.com/google/cloudprober/blob/master/rds/kubernetes/proto/config.proto#L38
    kubernetes_config {
      endpoints {}
    }
  }
}
```

This config adds a probe for endpoints named 'cloudprober'. Kubernetes targets configuration is further explained in the section below.

### Kubernetes RDS Targets

As explained [here](/concepts/targets/#resource-discovery-service), cloudprober uses RDS for dynamic targets discovery. In the above config, we add an internal RDS server that provides expansion for kubernetes `endpoints` (other supported types are -- _pods_, _services_).  Inside the probe, we specify targets of the type [rds_targets](/concepts/targets/#resource-discovery-service) with resource path, `k8s://endpoints/cloudprober`. This resource path specifies resource of the type 'endpoints' and with the name 'cloudprober' (Hint: you can skip the name part of the resource path to discover all endpoints in the cluster).

### Cluster Resources Access

RDS server that we added above discovers cluster resources using kubernetes APIs. It assumes that we are interested in the cluster we are running it in, and uses in-cluster config to talk to the kubernetes API server. For this set up to work, we need to give our container read-only access to kubernetes resources:

```yaml
# Define a ClusterRole (resource-reader) for read-only access to the cluster
# resources and bind this ClusterRole to the default service account.

cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
  name: resource-reader
  namespace: default
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
 name: default-resource-reader
 namespace: default
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
roleRef:
 kind: ClusterRole
 name: resource-reader
 apiGroup: rbac.authorization.k8s.io
EOF
```

This will give `default` service account read-only access to the cluster resources. If you don't want to give the "default" user this access, you can create a new service account for cloudprober and use it in the deployment spec above.

### Push Config Update

To push new cloudprober config to the cluster:

```bash
# Update the config map
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg  -o yaml --dry-run | \
  kubectl replace -f -

# Update deployment
CONFIG_CHECKSUM=$(kubectl get cm/cloudprober-config -o yaml | sha256sum) && \
cat deployment.yaml | envsubst | kubectl apply -f -
```

Cloudprober should now start monitoring cloudprober endpoints. To verify:

```bash
# Set up port fowarding such that you can access cloudprober:9313 through
# localhost:9313.
kubectl port-forward svc/cloudprober 9313:9313 &

# Check config
curl localhost:9313/config

# Check metrics
curl localhost:9313/metrics
```

If you're running on GKE and have not disabled cloud logging, you'll also see logs in [Stackdriver Logging](https://pantheon.corp.google.com/logs/viewer?resource=gce_instance).