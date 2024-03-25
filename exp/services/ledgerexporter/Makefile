SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# https://github.com/opencontainers/image-spec/blob/master/annotations.md
BUILD_DATE := $(shell date -u +%FT%TZ)
VERSION ?= $(shell git rev-parse --short HEAD)
DOCKER_IMAGE := stellar/ledger-exporter

docker-build:
	cd ../../../ && \
	$(SUDO) docker build --platform linux/amd64 --pull --label org.opencontainers.image.created="$(BUILD_DATE)" \
	--build-arg VERSION=$(VERSION) \
$(if $(STELLAR_CORE_VERSION), --build-arg STELLAR_CORE_VERSION=$(STELLAR_CORE_VERSION)) \
	-f exp/services/ledgerexporter/docker/Dockerfile \
	-t $(DOCKER_IMAGE):$(VERSION) \
	-t $(DOCKER_IMAGE):latest .

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
		 $(DOCKER_IMAGE):$(VERSION)

	$(SUDO) docker stop fake-gcs-server
	$(SUDO) docker rm fake-gcs-server
	$(SUDO) rm -rf ${PWD}/storage
	$(SUDO) docker network rm test-network

docker-push:
	$(SUDO) docker push $(DOCKER_IMAGE):$(VERSION)
	$(SUDO) docker push $(DOCKER_IMAGE):latest
