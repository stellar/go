# Check if we need to prepend docker commands with sudo
SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# If TAG is not provided set default value
ifeq ($(TAG),)
COMMIT := $(shell git rev-parse --short HEAD)
TAG := stellar/keystore:$(COMMIT)
endif

docker-build:
	cd ../../ && \
	$(SUDO) docker build -f services/keystore/Dockerfile -t $(TAG) .

docker-push:
	cd ../../ && \
	$(SUDO) docker push $(TAG)

docker: docker-build docker-push
