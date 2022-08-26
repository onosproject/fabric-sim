// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"testing"
)

// TestAccessFabricLoad loads simulator with the access.yaml topology and validates proper startup
func (s *TestSuite) TestAccessFabricLoad(t *testing.T) {
	devices := LoadAndValidate(t, "topologies/access.yaml", 3+6, (3*3*6+3*2)*2, 3*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)

	ProbeAllDevices(t, devices)
}
