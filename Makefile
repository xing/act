.PHONY: patch
patch:
	./apply-patches.sh

.PHONY: act-build
act-build: patch
	$(MAKE) -C act build

.PHONY: act-test
act-test: patch
	cd act && go test -v -timeout 25m ./...

.PHONY: clean
clean:
	git submodule foreach 'git am --abort || true'
	git submodule update --init --force --checkout

.PHONY: update-act
update-act:
	git submodule foreach git pull origin master
