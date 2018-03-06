VERSION ?= $(shell git describe --tags)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')

test:
	go test -v -race -covermode=atomic ./...

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o cloudprober -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

docker_build: $(BINARY) Dockerfile
	cp $(CACERTS) .
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(VERSION) .

docker_push:
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	docker login -u ${DOCKER_USER} -p ${DOCKER_PASS}
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

clean:
	rm cloudprober
	go get -u ./...
