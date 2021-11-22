.PHONY: patch
patch:
	./apply-patches.sh

.PHONY: act-build
act-build: patch
	$(MAKE) -C act build

.PHONY: act-test
act-test: patch
	cd act && go test -v ./...
