VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
DOCKER_IMAGE ?= cloudprober/cloudprober-contrib
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')

test:
        go test -v -race -covermode=atomic ./...

cloudprober:
        make -C .. cloudprober
        cp ../cloudprober .

bigquery_probe: $(SOURCES)
        CGO_ENABLED=0 go build -o bigquery_probe -ldflags "-X main.version=$(VERSION) -extldflags -static" ./gcp/cmd/bigquery_probe.go

ca-certificates.crt: $(CACERTS)
        cp $(CACERTS) ca-certificates.crt

docker_build: cloudprober bigquery_probe ca-certificates.crt Dockerfile
        docker build \
                --build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
                --build-arg VERSION=$(VERSION) \
                --build-arg VCS_REF=$(GIT_COMMIT) \
                -t $(DOCKER_IMAGE)  .
        docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)

docker_push:
        docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
        docker push $(DOCKER_IMAGE):$(VERSION)
        docker push $(DOCKER_IMAGE):latest

clean:
        rm cloudprober bigquery_probe
        go get -u ./...
