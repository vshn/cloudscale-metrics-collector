FROM docker.io/library/alpine:3.16 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

# TODO: Adjust binary file name
ENTRYPOINT ["cloudscale-metrics-collector"]
COPY cloudscale-metrics-collector /usr/bin/

USER 65536:0
