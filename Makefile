# SPDX-FileCopyrightText: 2022-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

FABRIC_SIM_VERSION ?= latest

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then mkdir -p build && cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

mod-update: # @HELP Download the dependencies to the vendor folder
	go mod tidy
	go mod vendor

mod-lint: mod-update # @HELP ensure that the required dependencies are in place
	# dependencies are vendored, but not committed, go.sum is the only thing we need to check
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

local-deps: # @HELP imports local deps in the vendor folder
local-deps: local-helmit local-onos-api local-onos-lib-go local-onos-test

build: # @HELP build the Go binaries and run all validations (default)
build: mod-update local-deps
	go build -mod=vendor -o build/_output/fabric-sim ./cmd/fabric-sim

test: # @HELP run the unit tests and source code validation producing a golang style report
test: mod-lint build linters license
	go test -race github.com/onosproject/fabric-sim/...

jenkins-test: # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: jenkins-tools mod-lint build linters license
	TEST_PACKAGES=github.com/onosproject/fabric-sim/... ./build/build-tools/build/jenkins/make-unit

integration-tests:  # @HELP run helmit integration tests locally
	(kubectl delete ns test || exit 0) && kubectl create ns test && helmit test -n test -c . ./cmd/fabric-sim-tests

cit:  # @HELP run helmit integration test under current development
	(kubectl delete ns test || exit 0) && kubectl create ns test && helmit test -n test -c . ./cmd/fabric-sim-tests --suite onoslite --test TestLiteONOSWithPlainMaxFabric

topologies:	# @HELP generate all topologies related artifacts
topologies: recipes netcfg robot

recipes:	# HELP generate topology files from the topology recipes
	@for topo in "plain_small" "plain_mid" "plain_large" "plain_max" "access" "access2" "superspine" "max"; do go run cmd/fabric-sim-topo/fabric-sim-topo.go gen topo --recipe topologies/$${topo}_recipe.yaml --output topologies/$${topo}.yaml; done

netcfg:	# HELP generate netcfg files from the topology files
	@for topo in "trivial" "plain_small" "plain_mid" "plain_large" "plain_max" "access" "access2" "superspine" "max"; do go run cmd/fabric-sim-topo/fabric-sim-topo.go gen netcfg --topology topologies/$${topo}.yaml --output topologies/$${topo}_netcfg.json; done

robot:	# HELP generate Robot test topology files from the topology files
	@for topo in "trivial" "plain_small" "plain_mid" "plain_large" "plain_max" "access" "access2" "superspine" "max"; do go run cmd/fabric-sim-topo/fabric-sim-topo.go gen robot --topology topologies/$${topo}.yaml --output topologies/$${topo}_robot.yaml; done

fabric-sim-docker: mod-update local-deps # @HELP build fabric-sim base Docker image
	docker build --platform linux/amd64 . -f build/fabric-sim/Dockerfile \
		-t ${DOCKER_REPOSITORY}fabric-sim:${FABRIC_SIM_VERSION}

images: # @HELP build all Docker images
images: fabric-sim-docker

docker-push-latest: docker-login
	docker push onosproject/fabric-sim:latest

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images kind-only

kind-only: # @HELP deploy the image without rebuilding first
kind-only:
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${DOCKER_REPOSITORY}fabric-sim:${FABRIC_SIM_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/fabric-sim

jenkins-publish: images docker-push-latest # @HELP Jenkins calls this to publish artifacts
	./build/build-tools/release-merge-commit
	./build/build-tools/build/docs/push-docs

clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/fabric-sim/fabric-sim ./cmd/onos/onos
	go clean -testcache github.com/onosproject/fabric-sim/...

local-helmit: # @HELP Copies a local version of the helmit dependency into the vendor directory
ifdef LOCAL_HELMIT
	rm -rf vendor/github.com/onosproject/helmit/go
	mkdir -p vendor/github.com/onosproject/helmit/go
	cp -r ${LOCAL_HELMIT}/go/* vendor/github.com/onosproject/helmit/go
endif

local-onos-api: # @HELP Copies a local version of the onos-api dependency into the vendor directory
ifdef LOCAL_ONOS_API
	rm -rf vendor/github.com/onosproject/onos-api/go
	mkdir -p vendor/github.com/onosproject/onos-api/go
	cp -r ${LOCAL_ONOS_API}/go/* vendor/github.com/onosproject/onos-api/go
endif

local-onos-lib-go: # @HELP Copies a local version of the onos-lib-go dependency into the vendor directory
ifdef LOCAL_ONOS_LIB_GO
	rm -rf vendor/github.com/onosproject/onos-lib-go
	mkdir -p vendor/github.com/onosproject/onos-lib-go
	cp -r ${LOCAL_ONOS_LIB_GO}/* vendor/github.com/onosproject/onos-lib-go
endif

local-onos-test: # @HELP Copies a local version of the onos-test dependency into the vendor directory
ifdef LOCAL_ONOS_TEST
	rm -rf vendor/github.com/onosproject/onos-test
	mkdir -p vendor/github.com/onosproject/onos-test
	cp -r ${LOCAL_ONOS_TEST}/* vendor/github.com/onosproject/onos-test
endif