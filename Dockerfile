FROM gcr.io/distroless/static-debian12
COPY bin/metallb-health-sidecar /metallb-health-sidecar
ENTRYPOINT ["/metallb-health-sidecar"]
