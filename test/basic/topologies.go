// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"strings"
	"testing"
)

// TestPlainFabricLoad loads simulator with the plain_mid.yaml topology and validates proper startup
func (s *TestSuite) TestPlainFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/plain_mid.yaml", 2+4, (3*2*4)*2, 4*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 1 })
	defer CleanUp(t)
	ProbeAllDevices(t, devices)
}

// TestPodFabricLoad loads simulator with the pod.yaml topology and validates proper startup
func (s *TestSuite) TestPodFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/pod.yaml", 2+6+6*12, (4*2*6+6*2*12)*2, 6*12*(20+1),
		func(d *simapi.Device) int {
			if strings.HasPrefix(string(d.ID), "spine") {
				return 64
			} else if strings.HasPrefix(string(d.ID), "leaf") {
				return 32
			}
			return 20 + 1 + 2 // IPU
		}, func(*simapi.Host) int { return 1 })
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

// TestFixedFabricLoad loads simulator with the fixed_fabric.yaml topology and validates proper startup
func (s *TestSuite) TestFixedFabricLoad(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/fixed_fabric.yaml", 14, 2*136, 40,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)
	ProbeAllDevices(t, devices)
}
