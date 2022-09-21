// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGenerateNetcfg(t *testing.T) {
	err := GenerateNetcfg("../../topologies/plain.yaml", "/tmp/plain_netcfg.json", "stratum-driver", "stratum-pipeconf")
	defer os.Remove("/tmp/plain_netcfg.json")
	assert.NoError(t, err)
}
