FROM alpine:3.18
COPY bin/metallb-health-sidecar /metallb-health-sidecar
ENTRYPOINT ["/metallb-health-sidecar"]
