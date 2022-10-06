// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"testing"
)

// TestPlainFabricLoad loads simulator with the plain.yaml topology and validates proper startup
func (s *TestSuite) TestPlainFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/plain.yaml", 2+4, (3*2*4)*2, 4*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 1 })
	defer CleanUp(t)
	ProbeAllDevices(t, devices)
}

// TestAccessFabricLoad loads simulator with the access.yaml topology and validates proper startup
func (s *TestSuite) TestAccessFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/access.yaml", 3+6, (3*3*6+3*2)*2, 3*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)
	ProbeAllDevices(t, devices)
}

// TestSuperSpineFabricLoad loads simulator with the superspine.yaml topology and validates proper startup
func (s *TestSuite) TestSuperSpineFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/superspine.yaml", 14, 2*136, 40,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)
	ProbeAllDevices(t, devices)
}
