---
menu:
    main:
        parent: "Exporting Metrics (Surfacers)"
        weight: 30
title: "Cloudwatch (AWS Cloud Monitoring)"
date: 2021-04-04T19:37:00+01:00
---
Cloudprober can natively export metrics to AWS Cloudwatch using the cloudwatch [surfacer](/surfacers/overview). Adding the cloudwatch surfacer to cloudprover is as simple as adding the following stanza to the config:

```
surfacer {
  type: CLOUDWATCH
}
```

## Authentication

The cloudwatch surfacer uses the AWS Go SDK, and supports the [default credential chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html):

1. Environment variables.
2. Shared credentials file.
3. If your application uses an ECS task definition or RunTask API operation, IAM role for tasks.
4. If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

## Authorization

In order to permit cloudprober to publish metric data to cloudwatch, ensure the profile being used for authentication has the following permissions, where the "cloudwatch:namespace" is the [metric namespace](#metric-namespace) used by cloudprober.

If the default metric namespace is changed, also change the condition in the IAM policy below to match the same value.

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Condition": {
                "StringEqualsIgnoreCase": {
                    "cloudwatch:namespace": "cloudprober"
                }
            },
            "Action": [
                "cloudwatch:PutMetricData"
            ],
            "Resource": [
                "*"
            ],
            "Effect": "Allow",
            "Sid": "PutMetrics"
        }
    ]
}
```

## [Metric Namespace](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/cloudwatch_concepts.html#Namespace)

The metric namespace used to publish metrics to by default is set to `cloudprober`. This can be changed by expanding the surfacer configuration:

```
surfacer {
  type: CLOUDWATCH

  cloudwatch_surfacer {
    namespace: "/cloudprober/website/probes"
  }
}
```

Note: If the namespace is modified, also modify the [IAM policy condition](#authorization) for the namespace PutMetricData call.

## Configuration Options

The full list of configuration options for the cloudwatch surfacer is:

```protobuf
  // The cloudwatch metric namespace
  optional string namespace = 1 [default = "cloudprober"];

  // The cloudwatch resolution value, lowering this below 60 will incur
  // additional charges as the metrics will be charged at a high resolution rate.
  optional int64 resolution = 2 [default=60];

  // If allowed_metrics_regex is specified, only metrics matching the given
  // regular expression will be exported to cloudwatch. Since probe type and
  // probe name are part of the metric name, you can use this field to restrict
  // cloudwatch metrics to a particular probe.
  // Example:
  // allowed_metrics_regex: ".*(http|ping).*(success|validation_failure).*"
  optional string allowed_metrics_regex = 3;
```

(Source: https://github.com/google/cloudprober/blob/master/surfacers/cloudwatch/proto/config.proto)

## Calculating the metric delta with Cloudwatch Metric Maths

The metrics produced by cloudprober are cumulative. Most services producing metrics into cloudwatch produce snapshot data whereby the metrics are recorded for a specific point in time.

In order to achieve a similar effect here, the [Cloudwatch Metric Maths](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/using-metric-math.html) RATE and PERIOD functions can be used to determine the delta values.

```
RATE(m1) * PERIOD(m1)
```

Whereby m1 is the metric id for the cloudprober metrics, for example:

```
namespace: cloudprober
metric name: latency
dst: google.com
ptype: http
probe: probe name
```
