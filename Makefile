VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')

test:
	go test -v -race -covermode=atomic ./...

GO_FLAGS=-ldflags "-X main.version=$(VERSION) -extldflags -static" 

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o cloudprober $(GO_FLAGS) ./cmd/cloudprober.go

ca-certificates.crt: $(CACERTS)
	cp $(CACERTS) ca-certificates.crt

docker_build: $(BINARY) ca-certificates.crt Dockerfile
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE)  .

docker_push:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker push $(DOCKER_IMAGE):$(DOCKER_VERSION)

docker_push_tagged:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker image push --all-tags $(DOCKER_IMAGE)

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

clean:
	rm -f cloudprober
	rm -rf ./builds
	go get -u ./...

build-all-platforms:
	$(MAKE) GOOS=linux   GOARCH=amd64       ./builds/linux/amd64/$(BINARY)
	$(MAKE) GOOS=linux   GOARCH=arm64       ./builds/linux/arm64/$(BINARY)
	$(MAKE) GOOS=linux   GOARCH=arm GOARM=7 ./builds/linux/arm/v7/$(BINARY)
	$(MAKE) GOOS=darwin  GOARCH=amd64       ./builds/darwin/amd64/$(BINARY)
	$(MAKE) GOOS=windows GOARCH=amd64       ./builds/windows/amd64/$(BINARY)

./builds/$(GOOS)/$(GOARCH)/$(BINARY):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GO_FLAGS) -o ./builds/$(GOOS)/$(GOARCH)/$(BINARY) ./cmd/cloudprober.go

./builds/$(GOOS)/$(GOARCH)/v$(GOARM)/$(BINARY):
	GCO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build $(GO_FLAGS) -o ./builds/$(GOOS)/$(GOARCH)/v$(GOARM)/$(BINARY) ./cmd/cloudprober.go


IMAGE_TAG?=$(DOCKER_IMAGE):$(DOCKER_VERSION)
PLATFORMS?=linux/arm/v7,linux/arm64/v8,linux/amd64
BUILDX_EXTRA_ARGS?=

push_args=--push $(BUILDX_EXTRA_ARGS)
build_args=$(BUILDX_EXTRA_ARGS)

.PHONY: _docker-%
_docker-%: build-all-platforms ca-certificates.crt
	docker buildx build --platform $(PLATFORMS) \
	--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
	--build-arg VERSION=$(VERSION) \
	--build-arg VCS_REF=$(GIT_COMMIT) \
	--tag $(IMAGE_TAG) \
	--tag $(DOCKER_IMAGE):latest \
	-f ./Dockerfile.multiarch \
	$($*_args) \
	.

.PHONY: docker-build
docker-build: _docker-build

.PHONY: docker-push
docker-push: _docker-push
