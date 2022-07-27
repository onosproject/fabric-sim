// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/loader"
	utils "github.com/onosproject/fabric-sim/test/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestLLDPPacket tests the LLDP packet-out handling
func (s *TestSuite) TestLLDPPacket(t *testing.T) {
	t.Logf("Creating fabric-sim connection")
	conn, err := utils.CreateConnection()
	assert.NoError(t, err)
	defer conn.Close()

	err = loader.LoadTopology(conn, "topologies/trivial.yaml")
	assert.NoError(t, err)
	defer CleanUp()

	deviceService := simapi.NewDeviceServiceClient(conn)

	// Get each of our two devices
	ctx := context.Background()
	r1, err := deviceService.GetDevice(ctx, &simapi.GetDeviceRequest{ID: simapi.DeviceID("switch1")})
	assert.NoError(t, err)
	r2, err := deviceService.GetDevice(ctx, &simapi.GetDeviceRequest{ID: simapi.DeviceID("switch2")})
	assert.NoError(t, err)

	// Create two stream listeners on switch2
	p4sw1, conn1 := GetP4Client(t, r1.Device)
	assert.NotNil(t, conn1)
	p4sw2a, conn2a := GetP4Client(t, r2.Device)
	assert.NotNil(t, conn2a)
	p4sw2b, conn2b := GetP4Client(t, r2.Device)
	assert.NotNil(t, conn2b)

	stream1 := StartStream(ctx, t, p4sw1, &p4api.Uint128{High: 0, Low: 1}, 0)
	stream2a := StartStream(ctx, t, p4sw2a, &p4api.Uint128{High: 0, Low: 1}, 0)
	stream2b := StartStream(ctx, t, p4sw2b, &p4api.Uint128{High: 0, Low: 1}, 7)

	// Create an LLDP packet
	lldp := layers.LinkLayerDiscovery{
		ChassisID: layers.LLDPChassisID{
			Subtype: layers.LLDPChassisIDSubTypeLocal,
			ID:      []byte("0"),
		},
		PortID: layers.LLDPPortID{
			Subtype: layers.LLDPPortIDSubtypeLocal,
			ID:      []byte("1024"),
		},
		TTL:    0,
		Values: nil,
	}

	buffer := gopacket.NewSerializeBuffer()
	err = lldp.SerializeTo(buffer, gopacket.SerializeOptions{})
	assert.NoError(t, err)

	err = stream1.Send(&p4api.StreamMessageRequest{
		Update: &p4api.StreamMessageRequest_Packet{
			Packet: &p4api.PacketOut{
				Payload: buffer.Bytes(),
			}},
	})
	assert.NoError(t, err)

	t.Log("Waiting for packet in on stream switch2a")
	msg2a, err := stream2a.Recv()
	assert.NoError(t, err)
	assert.NotNil(t, msg2a.GetPacket())
	packet := gopacket.NewPacket(msg2a.GetPacket().Payload, layers.LayerTypeLinkLayerDiscovery, gopacket.Default)
	assert.NotNil(t, packet)
	assert.NotNil(t, packet.Layer(layers.LayerTypeLinkLayerDiscovery))
	t.Log("Got LLDP packet on stream switch2a")

	t.Log("Waiting for packet in on stream switch2b")
	msg2b, err := stream2b.Recv()
	assert.NoError(t, err)
	assert.NotNil(t, msg2b.GetPacket())
	packet = gopacket.NewPacket(msg2b.GetPacket().Payload, layers.LayerTypeLinkLayerDiscovery, gopacket.Default)
	assert.NotNil(t, packet)
	assert.NotNil(t, packet.Layer(layers.LayerTypeLinkLayerDiscovery))
	t.Log("Got LLDP packet on stream switch2b")
}

// StartStream opens a new stream using the specified client and negotiates mastership using the supplied election ID
// Then it returns the new stream client.
func StartStream(ctx context.Context, t *testing.T, client p4api.P4RuntimeClient, electionID *p4api.Uint128, electionCode int32) p4api.P4Runtime_StreamChannelClient {
	stream, err := client.StreamChannel(ctx)
	assert.NoError(t, err)

	err = stream.Send(utils.CreateMastershipArbitration(electionID))
	assert.NoError(t, err)

	msg, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, electionCode, msg.GetArbitration().Status.Code)

	return stream
}
