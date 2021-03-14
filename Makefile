.DEFAULT_GOAL := help

ROOT_DIR       := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUILD_DIR      = $(ROOT_DIR)/bin
BUILD_FLAGS    ?=
BRANCH         = $(shell git rev-parse --abbrev-ref HEAD)
REVISION       = $(shell git describe --tags --always --dirty)
BUILD_DATE     = $(shell date +'%Y.%m.%d-%H:%M:%S')
LDFLAGS        ?= -w -s

BINARY        ?= firebox
LOCAL_IMAGE   ?= local/$(BINARY)

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: build
build: ## Build executables
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GO111MODULE=on go build -mod=vendor -o $(BUILD_DIR)/$(BINARY) -ldflags "$(LDFLAGS)" .

.PHONY: test
test: ## Test
	GO111MODULE=on go test -mod=vendor -v ./...

.PHONY: fmt
fmt: ## Go format
	go fmt ./...

.PHONY: clean
clean: ## Clean
	@rm -rf $(BUILD_DIR)

.PHONY: deps
deps: ## Get dependencies
	GO111MODULE=on go get ./...

.PHONY: vendor
vendor: ## Go vendor
	GO111MODULE=on go mod vendor

.PHONY: tidy
tidy: ## Go tidy
	GO111MODULE=on go mod tidy

.PHONY: docker-build
docker-build: ## Build docker image
	docker build -f Dockerfile -t $(LOCAL_IMAGE) .

.PHONY: docker-build-echo
docker-build-echo: ## Build docker image running server echo
	docker build -f Dockerfile.echo -t $(LOCAL_IMAGE)-echo .


SWAGGER_VERSION := v0.26.1
SWAGGER := docker run -u $(shell id -u):$(shell id -g) --rm -v $(CURDIR):$(CURDIR) -w $(CURDIR) -e GOCACHE=/tmp/.cache --entrypoint swagger quay.io/goswagger/swagger:$(SWAGGER_VERSION)

.PHONY: generate-server-api
generate-server-api: api/swagger.yaml ## Generate server API
	@echo GEN api/swagger.yaml
	$(SWAGGER) generate server -s server -a restapi \
			-t api \
			-f api/swagger.yaml \
			--exclude-main \
			--default-scheme=http

.PHONY: import-image
import-image: ## Import local image with echo server built by 'docker-build-echo' make target
	@sudo ignite image import --runtime docker $(LOCAL_IMAGE)-echo
	@sudo ignite image ls --log-level=error | grep "$(LOCAL_IMAGE)-echo" | awk '{ print "/var/lib/firecracker/image/"$$1"/image.ext4" }'
	$(eval IMAGE_FILE := $(shell sudo ignite image ls --log-level=error | grep "$(LOCAL_IMAGE)-echo" | awk '{ print "/var/lib/firecracker/image/"$$1"/image.ext4" }'))
	sudo cp $(IMAGE_FILE) $(ROOT_DIR)
	sudo chmod a+rw $(ROOT_DIR)/image.ext4

.PHONY: import-kernel
import-kernel: ## Import ignite-kernel
	@sudo ignite kernel import weaveworks/ignite-kernel:5.4.43
	@sudo ignite kernel ls --log-level=error | grep "weaveworks/ignite-kernel:5.4.43" | awk '{ print "/var/lib/firecracker/kernel/"$$1"/vmlinux" }'
	$(eval KERNEL_FILE := $(shell sudo ignite kernel ls --log-level=error | grep "weaveworks/ignite-kernel:5.4.43" | awk '{ print "/var/lib/firecracker/kernel/"$$1"/vmlinux" }'))
	sudo cp $(KERNEL_FILE) $(ROOT_DIR)
	sudo chmod a+rw $(ROOT_DIR)/vmlinux
