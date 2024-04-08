# Docker build targets use an optional "TAG" environment
# variable can be set to use custom tag name. For example:
#   TAG=my-registry.example.com/keystore:dev make keystore
DOWNLOADABLE_XDRS = xdr/Stellar-SCP.x \
xdr/Stellar-ledger-entries.x \
xdr/Stellar-ledger.x \
xdr/Stellar-overlay.x \
xdr/Stellar-transaction.x \
xdr/Stellar-types.x \
xdr/Stellar-contract-env-meta.x \
xdr/Stellar-contract-meta.x \
xdr/Stellar-contract-spec.x \
xdr/Stellar-contract.x \
xdr/Stellar-internal.x \
xdr/Stellar-contract-config-setting.x

XDRS = $(DOWNLOADABLE_XDRS) xdr/Stellar-lighthorizon.x \
		xdr/Stellar-exporter.x


XDRGEN_COMMIT=e2cac557162d99b12ae73b846cf3d5bfe16636de
XDR_COMMIT=b96148cd4acc372cc9af17b909ffe4b12c43ecb6

.PHONY: xdr xdr-clean xdr-update

keystore:
	$(MAKE) -C services/keystore/ docker-build

ticker:
	$(MAKE) -C services/ticker/ docker-build

friendbot:
	$(MAKE) -C services/friendbot/ docker-build

horizon:
	$(MAKE) -C services/horizon/ binary-build

ledger-exporter:
	$(MAKE) -C exp/services/ledgerexporter/ docker-build

webauth:
	$(MAKE) -C exp/services/webauth/ docker-build

recoverysigner:
	$(MAKE) -C exp/services/recoverysigner/ docker-build

regulated-assets-approval-server:
	$(MAKE) -C services/regulated-assets-approval-server/ docker-build

gxdr/xdr_generated.go: $(DOWNLOADABLE_XDRS)
	go run github.com/xdrpp/goxdr/cmd/goxdr -p gxdr -enum-comments -o $@ $(XDRS)
	gofmt -s -w $@

xdr/%.x:
	printf "%s" ${XDR_COMMIT} > xdr/xdr_commit_generated.txt
	curl -Lsf -o $@ https://raw.githubusercontent.com/stellar/stellar-xdr/$(XDR_COMMIT)/$(@F)

xdr/xdr_generated.go: $(DOWNLOADABLE_XDRS)
	docker run -it --rm -v $$PWD:/wd -w /wd ruby /bin/bash -c '\
		gem install specific_install -v 0.3.8 && \
		gem specific_install https://github.com/stellar/xdrgen.git -b $(XDRGEN_COMMIT) && \
		xdrgen \
			--language go \
			--namespace xdr \
			--output xdr/ \
			$(XDRS)'
	# No, you're not reading the following wrong. Apperantly, running gofmt twice required to complete it's formatting.
	gofmt -s -w $@
	gofmt -s -w $@

xdr: gxdr/xdr_generated.go xdr/xdr_generated.go

xdr-clean:
	rm $(DOWNLOADABLE_XDRS) || true

xdr-update: xdr-clean xdr
