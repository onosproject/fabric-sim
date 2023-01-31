# SPDX-FileCopyrightText: 2022-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: basic onoslite

namespace: # @HELP recreate the test k8s namespace
	(kubectl delete ns test || exit 0) && kubectl create ns test

basic: # @HELP run helmit basic tests locally
	helmit test -n test ./fabric-sim-tests -c .. --suite basic --no-teardown

onoslite: # @HELP run helmit ONOS-lite tests locally
	helmit test -n test ./fabric-sim-tests -c .. --suite onoslite --no-teardown --timeout 15m

