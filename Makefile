.PHONY: all
all: test metallb-health-sidecar

.PHONY: metallb-health-sidecar
metallb-health-sidecar:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/metallb-health-sidecar *.go
	strip bin/metallb-health-sidecar

.PHONY: test
test:
	go test -cover ./...

.PHONY: clean
clean:
	rm -f bin/*
