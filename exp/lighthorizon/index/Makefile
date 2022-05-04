XDRS = xdr/LightHorizon-types.x

XDRGEN_COMMIT=3f6808cd161d72474ffbe9eedbd7013de7f92748

.PHONY: xdr clean update

xdr/xdr_generated.go: $(XDRS)
	docker run -it --rm -v $$PWD:/wd -w /wd ruby /bin/bash -c '\
		gem install specific_install -v 0.3.7 && \
		gem specific_install https://github.com/stellar/xdrgen.git -b $(XDRGEN_COMMIT) && \
		xdrgen \
			--language go \
			--namespace xdr \
			--output xdr/ \
			$(XDRS)'
	ls -lAh
	go fmt $@

xdr: xdr/xdr_generated.go

clean:
	rm ./xdr/xdr_generated.go || true

update: clean xdr
