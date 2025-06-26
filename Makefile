default: install

all: reviewable install

install:
	go install

COV_UNIT := $(PWD)/tmp/cover/unit
COV_E2E := $(PWD)/tmp/cover/e2e
COV_MERGED := $(PWD)/tmp/cover/merged

reviewable: tidy fmt lint build test

tidy:
	go mod tidy

fmt:
	golangci-lint fmt

lint:
	golangci-lint run

build:
	go build ./...

test:
	rm -rf tmp/cover
	mkdir -p $(COV_UNIT) $(COV_MERGED)
	go test -cover ./... -args -test.gocoverdir=$(COV_UNIT)
	go tool covdata merge -i $(COV_UNIT),$(COV_E2E) -o $(COV_MERGED)
	$(call render_coverage, $(COV_UNIT))
	$(call render_coverage, $(COV_E2E))
	$(call render_coverage, $(COV_MERGED))

define render_coverage
	go tool covdata textfmt -i $(1) -o $(1)/cover.out
	go tool cover -html $(1)/cover.out -o $(1)/cover.html
	go tool covdata percent -i $(1)
endef
