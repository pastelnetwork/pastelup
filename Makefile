# Variables
export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
#export GOROOT ?= dist/

# Applications name
CGO    ?= 0
BINARY ?= pastelup
TEST_IMG = pastel-test

# Version
VERSION = $(shell git describe --tag)

# Target build and specific extention name
#PLATFORMS ?= darwin/amd64 windows/amd64/.exe linux/amd64

# Macros to sub info from platforms
#temp = $(subst /, ,$@)
#os = $(word 1, $(temp))
#arch = $(word 2, $(temp))
#ext = $(word 3, $(temp))

arch = "amd64"
ifeq ($(OS),Windows_NT)
	os = "windows"
	ext = ".exe"
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		os = "linux"
		ext = ""
	endif
	ifeq ($(UNAME_S),Darwin)
		os = "darwin"
		ext = ""
	endif
endif

# GO build flags
LDFLAGS="-s -w -X github.com/pastelnetwork/gonode/common/version.version=$(VERSION)"
#
# Target build
#
#release: $(PLATFORMS)

release:
	go build $(GCFLAGS) -ldflags=$(LDFLAGS) -o $(BINARY) main.go
	strip -v $(BINARY) -o dist/$(BINARY)-$(os)-$(arch)$(ext)

#$(PLATFORMS):
#	CGO_ENABLED=$(CGO) GOOS=$(os) GOARCH=$(arch) go build  $(GCFLAGS) -ldflags=$(LDFLAGS) -o dist/$(BINARY)-$(os)-$(arch)$(ext) main.go
# #	upx dist/$(BINARY)-$(os)-$(arch)$(ext)

#.PHONY: release $(PLATFORMS)

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

# clear files generated from using run-dev-container with mounted workdir
clean-dev:
	rm -rf .bash_history
	rm -rf .cache/
	rm -rf .keras/
	rm -rf pastel_dupe_detection_service/
	rm -rf venv/

lint:
	gofmt -d -e .
	go vet -v ./...
	revive -config ./.circleci/revive.toml ./...
	staticcheck ./...

build-test-img:
	docker build -t $(TEST_IMG) -f ./test/Dockerfile .

test-walletnode:
	$(eval CONTAINER_NAME := "pastel-walletnode-test")
	$(eval SCRIPT := "test-walletnode.sh")
	docker rm $(CONTAINER_NAME) || true
	docker run \
		--name $(CONTAINER_NAME) \
		--mount type=bind,source=${PWD}/test/scripts/$(SCRIPT),target=/home/ubuntu/$(SCRIPT) \
		--entrypoint '/bin/bash' \
		$(TEST_IMG) \
		-c "./$(SCRIPT)"

test-local-supernode:
	$(eval CONTAINER_NAME := "pastel-local-supernode-test")
	$(eval SCRIPT := "test-local-supernode.sh")
	docker rm $(CONTAINER_NAME) || true
	docker run \
		--name $(CONTAINER_NAME) \
		--interactive \
		--mount type=bind,source=${PWD}/test/scripts/$(SCRIPT),target=/home/ubuntu/$(SCRIPT) \
		--expose=19933 \
		--entrypoint '/bin/bash' \
		$(TEST_IMG) \
		-c "./$(SCRIPT)"

test-local-supernode-service:
	$(eval CONTAINER_NAME := "pastel-local-supernode-service-test")
	$(eval SCRIPT := "test-local-supernode.sh")
	docker rm $(CONTAINER_NAME) || true
	docker run \
		--name $(CONTAINER_NAME) \
		--interactive \
		--mount type=bind,source=${PWD}/test/scripts/$(SCRIPT),target=/home/ubuntu/$(SCRIPT) \
		--expose=19933 \
		--entrypoint '/bin/bash' \
		$(TEST_IMG) \
		-c "./$(SCRIPT) --enable-service"
	

test-ddservice:
	$(eval CONTAINER_NAME := "pastel-ddservice-test")
	$(eval SCRIPT := "test-ddservice.sh")
	docker rm $(CONTAINER_NAME) || true
	docker run -it \
		--name $(CONTAINER_NAME) \
		--mount type=bind,source=${PWD}/test/scripts/$(SCRIPT),target=/home/ubuntu/$(SCRIPT) \
		--entrypoint '/bin/bash' \
		$(TEST_IMG) \
		-c "./$(SCRIPT)"

