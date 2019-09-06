NAME=artefactor
AUTHOR=appvia
CONTAINER ?= quay.io/${AUTHOR}/${NAME}
AUTHOR_EMAIL=lewis.marshall@appvia.io
BINARY ?= ${NAME}
ROOT_DIR=${PWD}
HARDWARE=$(shell uname -m)
GIT_VERSION=$(shell git describe --always --tags --dirty)
GIT_SHA=$(shell git rev-parse HEAD)
GOVERSION=1.10
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
VERSION ?= ${GIT_VERSION}
PACKAGES=$(shell go list ./...)
GOFILES_NOVENDOR=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
VERSION_PKG=$(shell go list ./pkg/version)
LFLAGS ?= -X ${VERSION_PKG}.gitVersion=${GIT_VERSION} -X ${VERSION_PKG}.gitSha=${GIT_SHA}
VETARGS ?= -asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -unsafeptr
PLATFORMS ?= darwin linux
ARCHITECTURES ?= 386 amd64
E2E_BREAK_TESTS ?=

# E2E test values
MYSQL_IMAGE ?= mysql:5.7.27@sha256:1a121f2e7590f949b9ede7809395f209dd9910e331e8372e6682ba4bebcc020b
BUSYBOX_IMAGE ?= quay.io/google-containers/busybox:1.27.2@sha256:70892b4c36448ecd9418580da9efa2644c20b4401667db6ae0cf15c0dcdaa595
BUSYBOX2_IMAGE ?= busybox@sha256:442c9d8c2c01192d7f7a05a1b02ba0f4509d9bd28d4d40d67e8f7800740f1483
BUSYBOX3_IMAGE ?= busybox@sha256:2edbab3ccf5ebe2d1c79131966766ff2156df89ed538e0c8fb9a1f087b503a65
ALPINE_IMAGE ?= nginx:1-alpine
ARTEFACTOR_IMAGE_VARS ?= MYSQL_IMAGE ALPINE_IMAGE BUSYBOX_IMAGE BUSYBOX2_IMAGE BUSYBOX3_IMAGE
ARTEFACTOR_DOCKER_REGISTRY ?= localhost:5000

ARTEFACTOR_GIT_REPOS ?= .

export MYSQL_IMAGE BUSYBOX_IMAGE BUSYBOX2_IMAGE BUSYBOX3_IMAGE ALPINE_IMAGE ARTEFACTOR_IMAGE_VARS ARTEFACTOR_DOCKER_REGISTRY ARTEFACTOR_GIT_REPOS E2E_BREAK_TESTS

.PHONY: test authors changelog build release lint cover vet

default: build

golang:
	@echo "--> Go Version"
	@go version

build:
	@echo "--> Compiling the project"
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "${LFLAGS}" -o bin/${NAME} cmd/artefactor/*.go

release: clean release-deps
	@echo "--> Compiling all the static binaries"
	mkdir -p bin
	CGO_ENABLED=0 gox -arch="${ARCHITECTURES}" -os="${PLATFORMS}" -ldflags "-w ${LFLAGS}" -output=./bin/{{.Dir}}_{{.OS}}_{{.Arch}} ./...
	cd ./bin && sha256sum * > checksum.txt && cd -

docker_build:
	@echo "--> Creating a container"
	docker build . -t ${CONTAINER}:${VERSION}

run_e2e_test:
	@echo "--> running e2e test"
	ci_tests/e2etest.sh
	

run_updateimagevars_test:
	@echo "--> running update-image-vars test"
	@echo "env:" ${ARTEFACTOR_IMAGE_VARS}
	@env |grep IMAGE
	./bin/artefactor update-image-vars

docker_get_artefactor_binary:
	@echo "--> retreiveing artefactor binary"
	mkdir -p bin
	docker create \
    --name artefactor-build-${VERSION} \
    ${CONTAINER}:${VERSION} && \
	docker cp artefactor-build-${VERSION}:/usr/local/bin/artefactor bin/artefactor
	
docker_push:
	@echo "--> Pushing container"
	docker push ${CONTAINER}:${VERSION}

clean:
	rm -rf ./bin 2>/dev/null

authors:
	@echo "--> Updating the AUTHORS"
	git log --format='%aN <%aE>' | sort -u > AUTHORS


release-deps:
	@echo "--> Installing release dependencies"
	@GO111MODULE=off go get -u github.com/mitchellh/gox

vet:
	@echo "--> Running go vet $(VETARGS) ."
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		GO111MODULE=off go get golang.org/x/tools/cmd/vet; \
	fi
	@go vet $(VETARGS) $(PACKAGES)

lint:
	@echo "--> Running golint"
	@which golint 2>/dev/null ; if [ $$? -eq 1 ]; then \
		GO111MODULE=off go get -u golang.org/x/lint/golint; \
	fi
	@golint .

gofmt:
	@echo "--> Running gofmt check"
	@gofmt -s -l $(GOFILES_NOVENDOR) | grep -q \.go ; if [ $$? -eq 0 ]; then \
      echo "we have unformatted files - run 'make applygofmt' to apply"; \
			gofmt -s -d -l ${GOFILES_NOVENDOR}; \
      exit 1; \
    fi

applygofmt:
	@echo "--> Running gofmt apply"
	@gofmt -s -l -w $(GOFILES_NOVENDOR)

bench:
	@echo "--> Running go bench"
	@go test -v -bench=.

coverage:
	@echo "--> Running go coverage"
	@go test -cover $(PACKAGES) -coverprofile cover.out
	@go tool cover -html=cover.out -o cover.html

cover:
	@echo "--> Running go cover"
	@go test -cover $(PACKAGES)

test:
	@echo "--> Running the tests"
	@go test -v ${PACKAGES}
	@$(MAKE) cover

src:
	@echo "--> Running the src checks"
	@$(MAKE) vet
	@$(MAKE) lint
	@$(MAKE) gofmt

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B > changelog
