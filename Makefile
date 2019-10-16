# Builds keystore docker image. Optional "TAG" environment
# variable can be set to use custom tag name. For example:
#   TAG=my-registry.example.com/keystore:dev make keystore
keystore:
	$(MAKE) -C services/keystore/ docker-build
