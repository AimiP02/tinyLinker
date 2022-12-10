BIN_PATH = ./bin

TESTS := $(wildcard tests/*.sh)

build:
	go build -o $(BIN_PATH)

test: build
	$(MAKE) $(TESTS)
	@printf '\e[32mAll tests passed!\e[0m\n'

$(TESTS):
	@echo 'Testing' $@
	@./$@
	@printf '\e[32mOK\e[0m\n'

clean:
	rm -rf tests/out/* bin/*

.PHONY: build clean test $(TESTS)