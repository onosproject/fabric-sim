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
	t.Logf("Creating fabric-sim connection")
	conn, err := utils.CreateConnection()
	assert.NoError(t, err)
	defer conn.Close()

	err = topo.LoadTopology(conn, "topologies/custom.yaml")
	assert.NoError(t, err)
	defer CleanUp()

	// Validate that everything got loaded correctly
	deviceClient := simapi.NewDeviceServiceClient(conn)
	linksClient := simapi.NewLinkServiceClient(conn)
	hostClient := simapi.NewHostServiceClient(conn)

	t.Logf("Validating topology")

	// Do we have all the devices?
	ctx := context.Background()
	dr, err := deviceClient.GetDevices(ctx, &simapi.GetDevicesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, len(dr.Devices), 6)

	// Do we have all the links?
	lr, err := linksClient.GetLinks(ctx, &simapi.GetLinksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, len(lr.Links), 16)

	// Do we have all the hosts?
	hr, err := hostClient.GetHosts(ctx, &simapi.GetHostsRequest{})
	assert.NoError(t, err)
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
		p4Client, p4conn := GetP4Client(t, device)
		defer p4conn.Close()

		t.Logf("Getting P4 capabilities device %s", device.ID)
		cr, err := p4Client.Capabilities(ctx, &p4api.CapabilitiesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, cr.P4RuntimeApiVersion, "1.1.0")

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

// CleanUp cleans up the simulation to allow other simulation tests run
func CleanUp() {
	if conn, err := utils.CreateConnection(); err == nil {
		_ = topo.ClearTopology(conn)
	}
}
