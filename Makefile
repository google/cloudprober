VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GIT_BRANCH = $(strip $(shell git branch --show-current))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')

test:
	go test -v -race -covermode=atomic ./...

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o cloudprober -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

ca-certificates.crt: $(CACERTS)
	cp $(CACERTS) ca-certificates.crt

docker_build: $(BINARY) ca-certificates.crt Dockerfile
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE)  .

docker_push_master:
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker push $(DOCKER_IMAGE):$(GIT_BRANCH)

docker_push_tagged:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker image push --all-tags $(DOCKER_IMAGE)

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

clean:
	rm cloudprober
	go get -u ./...
