SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# https://github.com/opencontainers/image-spec/blob/master/annotations.md
BUILD_DATE := $(shell date -u +%FT%TZ)
VERSION ?= latest
TAG ?= stellar/ledger-exporter:$(VERSION)

docker-build:
ifndef STELLAR_CORE_VERSION
	cd ../../../ && \
	$(SUDO) docker build --pull --label org.opencontainers.image.created="$(BUILD_DATE)" \
	--build-arg VERSION=$(VERSION) \
	-f exp/services/ledgerexporter/docker/Dockerfile \
	-t $(TAG) .
else
	cd ../../../ && \
	$(SUDO) docker build --pull --label org.opencontainers.image.created="$(BUILD_DATE)" \
	--build-arg VERSION=$(VERSION) --build-arg STELLAR_CORE_VERSION=$(STELLAR_CORE_VERSION) \
	-f exp/services/ledgerexporter/docker/Dockerfile \
	-t $(TAG) .
endif

docker-test:
	# Create temp storage dir
	$(SUDO) mkdir -p ${PWD}/storage/exporter-test

	# Create test network for docker
	$(SUDO) docker network create test-network

	# Run the fake GCS server
	$(SUDO) docker run -d --name fake-gcs-server -p 4443:4443 \
		 -v ${PWD}/storage:/data --network test-network fsouza/fake-gcs-server -scheme http

	# Run the ledger-exporter
	$(SUDO) docker run --platform linux/amd64 -t --network test-network\
		 -e NETWORK=pubnet \
		 -e ARCHIVE_TARGET=gcs://exporter-test \
		 -e START=1000 \
		 -e END=2000 \
		 -e STORAGE_EMULATOR_HOST=http://fake-gcs-server:4443 \
		 stellar/ledger-exporter

	# Cleanup
	$(SUDO) docker stop fake-gcs-server
	$(SUDO) docker rm fake-gcs-server
	$(SUDO) rm -rf ${PWD}/storage
	$(SUDO) docker network rm test-network

docker-push:
	cd ../../../ && \
	$(SUDO) docker push $(TAG)
