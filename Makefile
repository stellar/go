# Docker build targets use an optional "TAG" environment
# variable can be set to use custom tag name. For example:
#   TAG=my-registry.example.com/keystore:dev make keystore

keystore:
	$(MAKE) -C services/keystore/ docker-build

webauth:
	$(MAKE) -C exp/services/webauth/ docker-build
