// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package onoslite

import (
	"github.com/onosproject/fabric-sim/test/basic"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestLiteONOS loads simulator with plain.yaml topology, creates lite ONOS controller and points it at the
// simulated topology
func (s *TestSuite) TestLiteONOS(t *testing.T) {
	devices := basic.LoadAndValidate(t, "topologies/plain.yaml", 6, 48, 80,
		func(device *simapi.Device) int { return 32 }, func(host *simapi.Host) int { return 1 })
	defer basic.CleanUp(t)

	onos := NewLiteONOS()

	pointers := extractDevicePointers(devices)
	err := onos.Start(pointers)
	assert.NoError(t, err)

	defer func() { _ = onos.Stop() }()

	time.Sleep(1 * time.Minute)

	t.Logf("Validating discovered topology...")

	// Did we discover all devices?
	assert.Len(t, onos.Devices, 6)

	// Do all devices have the right number of ports?
	for _, device := range onos.Devices {
		assert.Len(t, device.Ports, 32)
	}

	// Did we discover all links?
	assert.Len(t, onos.Links, 48)
	// Did we discover all hosts?
	assert.Len(t, onos.Hosts, 80)
}

func extractDevicePointers(devices []*simapi.Device) []*DevicePointer {
	pointers := make([]*DevicePointer, 0, len(devices))
	for _, device := range devices {
		pointers = append(pointers, &DevicePointer{ID: string(device.ID), ChassisID: device.ChassisID, ControlPort: device.ControlPort})
	}
	return pointers
}
