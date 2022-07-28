// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"testing"
)

// TestSuperSpineLoad loads simulator with the superspine_fabric.yaml topology and validates proper startup
func (s *TestSuite) TestSuperSpineLoad(t *testing.T) {
	devices := LoadAndValidate(t, "topologies/superspine_fabric.yaml", 14, 2*136, 40,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)

	ProbeAllDevices(t, devices)
}
