# Variables
export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
#export GOROOT ?= dist/

# Applications name
CGO    ?= 0
BINARY ?= pastel-utility

# Version
VERSION = $(shell git rev-parse --short=8 HEAD)

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

# upx dist/$(BINARY)-$(os)-$(arch)
$(PLATFORMS):
	CGO_ENABLED=$(CGO) GOOS=$(os) GOARCH=$(arch) go build $(GCFLAGS) -ldflags=$(LDFLAGS) -o dist/$(BINARY)-$(os)-$(arch)$(ext) main.go
	upx dist/$(BINARY)-$(os)-$(arch)$(ext)

.PHONY: release $(PLATFORMS)