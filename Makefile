# Variables
export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
#export GOROOT ?= dist/

# Applications name
CGO    ?= 0
BINARY ?= pastelup

# Version
VERSION = $(shell git describe --tag)

# Target build and specific extention name
PLATFORMS ?= darwin/amd64 windows/amd64/.exe linux/amd64

# Macros to sub info from platforms
temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))
ext = $(word 3, $(temp))

# GO build flags
LDFLAGS="-s -w -X github.com/pastelnetwork/gonode/common/version.version=$(VERSION)"
#
# Target build
#
release: $(PLATFORMS)

# upx dist/$(BINARY)-$(os)-$(arch)$(ext
$(PLATFORMS):
	CGO_ENABLED=$(CGO) GOOS=$(os) GOARCH=$(arch) go build  $(GCFLAGS) -ldflags=$(LDFLAGS) -o dist/$(BINARY)-$(os)-$(arch)$(ext) main.go
	#upx dist/$(BINARY)-$(os)-$(arch)$(ext)

.PHONY: release $(PLATFORMS)

build-dev-container:
	docker build -t pastel-dev -f ./test/Dockerfile-dev .

# useful if developing on a non-linux OS like a mac
run-dev-container:
	docker rm pastel-dev || true
	docker run -it \
		--name pastel-dev \
		--mount type=bind,source=${PWD},target=/home/ubuntu \
		--entrypoint '/bin/bash' \
		--memory="1g" \
		--memory-swap="2g" \
		pastel-dev

clean-dev:
	rm -rf .bash_history
	rm -rf .cache/
	rm -rf .keras/
	rm -rf pastel_dupe_detection_service/
	rm -rf venv/

build-test-img:
	docker build -t pastel-test -f ./test/Dockerfile .

test-walletnode:
	docker rm pastel-walletnode-test || true
	docker run \
		--name pastel-walletnode-test \
		--mount type=bind,source=${PWD}/test/scripts/test-walletnode.sh,target=/home/ubuntu/test-walletnode.sh \
		--entrypoint '/bin/sh' \
		pastel-test \
		-c "./test-walletnode.sh"

test-ddservice:
	docker rm pastel-ddservice-test || true
	docker run -it \
		--name pastel-ddservice-test \
		--mount type=bind,source=${PWD}/test/scripts/test-ddservice.sh,target=/home/ubuntu/test-ddservice.sh \
		--entrypoint '/bin/bash' \
		pastel-test \
		-c "./test-ddservice.sh"

