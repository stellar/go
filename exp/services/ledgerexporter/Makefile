SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# https://github.com/opencontainers/image-spec/blob/master/annotations.md
BUILD_DATE := $(shell date -u +%FT%TZ)
VERSION=latest
TAG ?= stellar/ledger-exporter:$(VERSION)

docker-build:
	cd ../../../ && \
	$(SUDO) docker build --pull --label org.opencontainers.image.created="$(BUILD_DATE)" \
    --build-arg VERSION=$(VERSION) --build-arg STELLAR_CORE_VERSION=$(STELLAR_CORE_VERSION) \
	-f exp/services/ledgerexporter/docker/Dockerfile \
	-t $(TAG) .

docker-push:
	cd ../../../ && \
	$(SUDO) docker push $(TAG)
