[![Build
Status](https://travis-ci.org/google/cloudprober.svg?branch=master)](https://travis-ci.org/google/cloudprober)

# Cloudprober

Cloudprober is a monitoring software that runs user-defined probes on a machine.
These probes can provide critical insights into the systems' avaialbility and
performance. Cloudprober comes with some built-in core probe types: PING (a fast
ping prober that can probe thousands of targets with minimal resources), HTTP,
UDP and DNS. More complex probing can be done through the EXTERNAL probe type
that allows using any arbitrary program for probing.

Cloudprober exports probe results as counter based metrics that work well with
Prometheus and Grafana. Cloudprober also has built-in support for StackDriver.
Support for more monitoring systems can be added with minimal effort.

Even though Cloudprober is a generic prober software, it's been created with
Cloud in mind and it provides some core features, such as auto-discovery of
Cloud targets (supports GCP resources out of the box, support for other Cloud
systems can be added easily), that make it really easy to use in Cloud.
