BIN_PATH = ./bin

VERSION = 0.1.0
COMMIT_ID = $(shell git rev-list -1 HEAD)
TESTS := $(wildcard tests/*.sh)

build:
	@go build -o $(BIN_PATH) -ldflags "-X main.version=${VERSION}-${COMMIT_ID}"
	@ln -sf ./bin/rvld ld

test: build
	@CC="riscv64-linux-gnu-gcc" \
	$(MAKE) $(TESTS)
	@printf '\e[32mAll tests passed!\e[0m\n'

$(TESTS):
	@echo 'Testing' $@
	@./$@
	@printf '\e[32mOK\e[0m\n'

clean:
	rm -rf tests/out/* bin/*

.PHONY: build clean test $(TESTS)
