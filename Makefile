.PHONY: run build clean coveralls cover install-gotestsum test-report release

.DEFAULT_GOAL := run

# Configuration for goreleaser
PACKAGE_NAME := github.com/bjartek/aether
GOLANG_CROSS_VERSION ?= v1.25.0

run: build
	cd example && ../aether  && cd ..

build:
	go build .

clean:
	rm -f aether 

coveralls:
	go test --timeout 120s -coverprofile=profile.cov -covermode=atomic -coverpkg=github.com/bjartek/aether -v ./...

cover: coveralls
	go tool cover -html=profile.cov

install-gotestsum:
	go install gotest.tools/gotestsum@latest

test-report: install-gotestsum
	gotestsum -f testname --no-color --hide-summary failed --junitfile test-result.xml

release:
	docker run \
		--rm \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean
