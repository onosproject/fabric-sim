// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/topo"
	utils "github.com/onosproject/fabric-sim/test/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"strings"
	"testing"
)

// TestTopologyLoad loads simulator with custom.yaml topology and validates proper startup
func (s *TestSuite) TestTopologyLoad(t *testing.T) {
	devices := LoadAndValidate(t, "topologies/custom.yaml", 6, 2*8, 16,
		spineAndLeafPorts, func(host *simapi.Host) int { return 1 })
	defer CleanUp(t)

	ProbeAllDevices(t, devices)
}

func spineAndLeafPorts(device *simapi.Device) int {
	if strings.Contains(string(device.ID), "spine") {
		return 4
	}
	return 8
}

// DevicePortCount returns the expected number of ports for a devuce
type DevicePortCount func(device *simapi.Device) int

// HostNICCount returns the expected number of NICs for a host
type HostNICCount func(host *simapi.Host) int

// LoadAndValidate loads the specified topology and validates the correct counts of devices, links and hosts
func LoadAndValidate(t *testing.T, path string, devices int, links int, hosts int, portsPerDevice DevicePortCount, nicsPerHost HostNICCount) []*simapi.Device {
	conn, err := utils.CreateConnection()
	assert.NoError(t, err)
	defer conn.Close()

	err = topo.ClearTopology(conn)
	assert.NoError(t, err)

	err = topo.LoadTopology(conn, path)
	assert.NoError(t, err)

	// Validate that everything got loaded correctly
	deviceClient := simapi.NewDeviceServiceClient(conn)
	linkClient := simapi.NewLinkServiceClient(conn)
	hostClient := simapi.NewHostServiceClient(conn)

	t.Logf("Validating topology")

	// Do we have all the devices?
	ctx := context.Background()
	dr, err := deviceClient.GetDevices(ctx, &simapi.GetDevicesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, devices, len(dr.Devices))

	// Do we have all the links?
	lr, err := linkClient.GetLinks(ctx, &simapi.GetLinksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, links, len(lr.Links))

	// Do we have all the hosts?
	hr, err := hostClient.GetHosts(ctx, &simapi.GetHostsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, hosts, len(hr.Hosts))

	// What about all the device ports?
	for _, device := range dr.Devices {
		assert.Equal(t, portsPerDevice(device), len(device.Ports))
	}

	// What about all the host NICs?
	for _, host := range hr.Hosts {
		assert.Equal(t, nicsPerHost(host), len(host.Interfaces))
	}
	return dr.Devices
}

// CleanUp cleans up the simulation
func CleanUp(t *testing.T) {
	t.Log("Cleaning up topology")
	if conn, err := utils.CreateConnection(); err == nil {
		if err := topo.ClearTopology(conn); err != nil {
			t.Log("Unable to clear topology")
			assert.NoError(t, err)
		}
	} else {
		t.Log("Unable to clear topology; no connection")
		t.Fail()
	}
}

// ProbeAllDevices tests each device P4Runtime agent port by requesting capabilities
func ProbeAllDevices(t *testing.T, devices []*simapi.Device) {
	ctx := context.Background()
	for _, device := range devices {
		t.Logf("Connecting to agent for device %s", device.ID)
		p4Client, p4conn := GetP4Client(t, device)
		defer p4conn.Close()

		t.Logf("Getting P4 capabilities device %s", device.ID)
		cr, err := p4Client.Capabilities(ctx, &p4api.CapabilitiesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "1.1.0", cr.P4RuntimeApiVersion)

		// Open message stream and negotiate mastership for default (no) role
		t.Logf("Negotiating mastership for device %s", device.ID)
		stream, err := p4Client.StreamChannel(ctx)
		assert.NoError(t, err)

		err = stream.Send(utils.CreateMastershipArbitration(&p4api.Uint128{High: 0, Low: 1}))
		assert.NoError(t, err)

		msg, err := stream.Recv()
		assert.NoError(t, err)
		assert.Equal(t, int32(0), msg.GetArbitration().Status.Code)
	}
}

// GetP4Client returns a new P4Runtime service client and its underlying connection for the given device
func GetP4Client(t *testing.T, device *simapi.Device) (p4api.P4RuntimeClient, *grpc.ClientConn) {
	conn, err := utils.CreateDeviceConnection(device)
	assert.NoError(t, err)
	return p4api.NewP4RuntimeClient(conn), conn
}
