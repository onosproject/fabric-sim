// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/loader"
	utils "github.com/onosproject/fabric-sim/test/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"strings"
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

	// Validate that everything got loaded correctly
	conn, err = utils.CreateConnection()
	assert.NilError(t, err)

	deviceService := simapi.NewDeviceServiceClient(conn)
	linksService := simapi.NewLinkServiceClient(conn)
	hostService := simapi.NewHostServiceClient(conn)

	t.Logf("Validating topology")

	// Do we have all the devices?
	ctx := context.Background()
	dr, err := deviceService.GetDevices(ctx, &simapi.GetDevicesRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(dr.Devices), 6)

	// Do we have all the links?
	lr, err := linksService.GetLinks(ctx, &simapi.GetLinksRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(lr.Links), 16)

	// Do we have all the hosts?
	hr, err := hostService.GetHosts(ctx, &simapi.GetHostsRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(hr.Hosts), 16)

	// What about all the host NICs?
	for _, host := range hr.Hosts {
		assert.Equal(t, len(host.Interfaces), 1)
	}

	// What about all the spine and leaf ports?
	for _, device := range dr.Devices {
		if strings.Contains(string(device.ID), "spine") {
			assert.Equal(t, len(device.Ports), 4)
		} else {
			assert.Equal(t, len(device.Ports), 8)
		}
	}

	// Test each device P4Runtime agent port by requesting capabilities
	for _, device := range dr.Devices {
		t.Logf("Connecting to agent for device %s", device.ID)
		p4rconn, err := utils.CreateDeviceConnection(device)
		assert.NilError(t, err)

		t.Logf("Getting P4 capabilities device %s", device.ID)
		p4Service := p4api.NewP4RuntimeClient(p4rconn)
		cr, err := p4Service.Capabilities(ctx, &p4api.CapabilitiesRequest{})
		assert.NilError(t, err)
		assert.Equal(t, cr.P4RuntimeApiVersion, "1.1.0")
	}
}
