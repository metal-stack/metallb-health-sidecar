FROM gcr.io/distroless/static-debian13
COPY bin/metallb-health-sidecar /metallb-health-sidecar
ENTRYPOINT ["/metallb-health-sidecar"]
