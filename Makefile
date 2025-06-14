PHONY: build test

build:
	cd cmd/beam && go build -o beam .

test:
	cd pkg/tcp && go test -v .