// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestSuperSpineLoad loads simulator with the superspine_fabric.yaml topology and validates proper startup
func (s *TestSuite) TestSuperSpineLoad(t *testing.T) {
	t.Skip("Investigating hang during the link load")
	devices := LoadAndValidate(t, "topologies/superspine_fabric.yaml", 14, 200, 40, 2)
	defer CleanUp(t)

	t.Logf("Validating device ports")

	// What about all the spine and leaf ports?
	for _, device := range devices {
		assert.Equal(t, 32, len(device.Ports))
	}
}
