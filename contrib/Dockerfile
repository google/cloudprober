# This Dockerfile packages contrib binaries alongwith cloudprober
# and ca certificates.
#
# Docker image built using this can executed in the following manner:
#   docker run -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
#                         cloudprober/cloudprober
FROM busybox
ADD cloudprober /cloudprober
ADD bigquery_probe /contrib/bigquery_probe
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Metadata params
ARG BUILD_DATE
ARG VERSION
ARG VCS_REF

# Metadata
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="Cloudprober" \
      org.label-schema.vcs-url="https://github.com/google/cloudprober" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.version=$VERSION \
      com.microscaling.license="Apache-2.0"

ENTRYPOINT ["/cloudprober", "--logtostderr"]
