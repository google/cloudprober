---
menu:
    main:
        parent: "How-Tos"
        weight: 35
title: "Built-in Servers"
date: 2019-10-11T17:48:13-05:00
---
Cloudprober has a few built in servers. This is useful when you are probing that
a connection is working, or as a baseline to compare the probing results from
your actual service to.

## HTTP

{{< highlight shell >}}
server {
  type: HTTP
  http_server {
    port: 8080
  }
}
{{< / highlight >}}

This creates an HTTP server that responds on port `8080`. By default it will
respond to the following endpoints:

* `/healthcheck`
* `/lameduck`

{{< highlight shell >}}
server {
  type: HTTP
  http_server {
    port: 8080
    pattern_data_handler {
      response_size: 1024
    }
    pattern_data_handler {
      response_size: 4
      pattern: "four"
    }
  }
}
{{< / highlight >}}

This adds two endpoints to the HTTP server:

*   `/data_1024` which responds with 1024 bytes of
    `cloudprobercloudprobercloudprober`.
*   `/data_4` which responds with `four`.

See
[servers/http/proto/config.go](https://github.com/google/cloudprober/blob/master/servers/http/proto/config.proto)
for all HTTP server configuration options.

## UDP

A Cloudprober UDP server can be configured to either echo or discard packets it
receives.

{{< highlight shell >}}
server {
  type: UDP
  udp_server {
    port: 85
    type: ECHO
  }
}

server {
  type: UDP
  udp_server {
    port: 90
    type: DISCARD
  }
}
{{< / highlight >}}

See
[servers/udp/proto/config.go](https://github.com/google/cloudprober/blob/master/servers/udp/proto/config.proto)
for all UDP server configuration options.

## GRPC

See
[servers/grpc/proto/config.go](https://github.com/google/cloudprober/blob/master/servers/grpc/proto/config.proto)
for all GRPC server configuration options.
