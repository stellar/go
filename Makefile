# Docker build targets use an optional "TAG" environment
# variable can be set to use custom tag name. For example:
#   TAG=my-registry.example.com/keystore:dev make keystore
XDRS = xdr/Stellar-SCP.x \
xdr/Stellar-ledger-entries.x \
xdr/Stellar-ledger.x \
xdr/Stellar-overlay.x \
xdr/Stellar-transaction.x \
xdr/Stellar-types.x

.PHONY: xdr

keystore:
	$(MAKE) -C services/keystore/ docker-build

ticker:
	$(MAKE) -C services/ticker/ docker-build

friendbot:
	$(MAKE) -C services/friendbot/ docker-build

webauth:
	$(MAKE) -C exp/services/webauth/ docker-build

recoverysigner:
	$(MAKE) -C exp/services/recoverysigner/ docker-build

regulated-assets-approval-server:
	$(MAKE) -C services/regulated-assets-approval-server/ docker-build

gxdr/xdr_generated.go: $(XDRS)
	go run github.com/xdrpp/goxdr/cmd/goxdr -p gxdr -enum-comments -o $@ $(XDRS)
	go fmt $@

xdr/%.x:
	curl -Lsf -o $@ https://raw.githubusercontent.com/stellar/stellar-core/master/src/$@

define xdrheader
//lint:file-ignore S1005 The issue should be fixed in xdrgen. Unfortunately, there's no way to ignore a single file in staticcheck.
//lint:file-ignore U1000 fmtTest is not needed anywhere, should be removed in xdrgen.
endef
export xdrheader

xdr/xdr_generated.go: $(XDRS)
	docker run --platform linux/amd64 -it --rm -v $$PWD:/wd -w /wd ruby /bin/bash -c '\
		gem install specific_install && \
		gem specific_install https://github.com/stellar/xdrgen.git -b master && \
		xdrgen \
			--language go \
			--namespace xdr \
			--output xdr/ \
			$(XDRS)'
	echo "$$xdrheader" > $@.tmp
	cat $@ >> $@.tmp
	mv $@.tmp $@
	go fmt $@

xdr: gxdr/xdr_generated.go xdr/xdr_generated.go
