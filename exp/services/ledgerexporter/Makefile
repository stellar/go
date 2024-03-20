SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# https://github.com/opencontainers/image-spec/blob/master/annotations.md
BUILD_DATE := $(shell date -u +%FT%TZ)
VERSION=test
TAG ?= stellar/ledger-exporter:$(VERSION)

docker-build:
	cd ../../../ docker build -f exp/services/ledgerexporter/docker/Dockerfile -t $(TAG) .

docker-push:
	cd ../../../ && \
	$(SUDO) docker push $(TAG)
