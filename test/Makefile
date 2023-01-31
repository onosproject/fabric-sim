# SPDX-FileCopyrightText: 2022-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

helmit-basic: # @HELP run helmit basic tests locally
	helmit test -n test ./fabric-sim-tests --suite basic --no-teardown

helmit-onoslite: # @HELP run helmit ONOS-lite tests locally
	helmit test -n test ./fabric-sim-tests --suite onoslite --no-teardown --timeout 15m

