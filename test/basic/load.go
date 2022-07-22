// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"github.com/onosproject/fabric-sim/pkg/loader"
	utils "github.com/onosproject/fabric-sim/test/utils"
	"testing"

	"gotest.tools/assert"
)

// TestTopologyLoad loads simulator with custom.yaml topology and validates proper startup
func (s *TestSuite) TestTopologyLoad(t *testing.T) {
	t.Logf("Creating fabric-sim connection")
	conn, err := utils.CreateConnection()
	assert.NilError(t, err)

	t.Logf("Loading topology")
	err = loader.LoadTopology(conn, "topologies/custom.yaml")
	assert.NilError(t, err)

	// TODO: add assertions about number of devices, hosts and links being created
	// TODO: test each device agent port
}
