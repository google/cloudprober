---
menu:
    main:
        parent: "How-Tos"
        weight: 20
title: "Running On Kubernetes"
date: 2019-10-08T17:24:32-07:00
---

Kubernetes is a popular platform for running containers, and Cloudprober container runs on Kubernetes right out of the box. This document shows how you can use config map to provide config to cloudprober and reload cloudprober on config changes.

In Kubernetes, a convenient way to provide config to containers is to use config maps.  Let's create a config that specifies a probe to monitor "google.com".

```bash
probe {
  name: "google-http"
  type: HTTP
  targets {
    host_names: "www.google.com"
  }
  http_probe {
      protocol: HTTP
      relative_url: "/"
  }
  interval_msec: 15000
  timeout_msec: 1000
}
```

Save this config in `cloudprober.cfg`, create a config map using the following command:

```bash
kubectl create configmap cloudprober-config \
  --from-file=cloudprober.cfg=cloudprober.cfg
```



Now let's add a `deployment.yaml` to add the config volume and cloudprober container:

```yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: cloudprober
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: cloudprober
        annotations:
          checksum/config: "${CONFIG_CHECKSUM}"
    spec:
      volumes:
      - name: cloudprober-config
        configMap:
          name: cloudprober-config
      containers:
      - name: cloudprober
        image: cloudprober/cloudprober
        command: ["/usr/bin/cloudprober"]
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
export CONFIG_CHECKSUM=$(sha256sum cloudprober.cfg); cat deployment.yaml | envsubst | \
  kubectl apply -f -
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