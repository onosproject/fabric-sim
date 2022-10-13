// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGenerateRobot(t *testing.T) {
	err := GenerateRobotTopology("../../topologies/plain_mid.yaml", "/tmp/robot.yaml")
	defer os.Remove("/tmp/robot.yaml")
	assert.NoError(t, err)
}
