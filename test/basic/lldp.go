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
	"strconv"
	"testing"
	"time"
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
			ID:      []byte("switch1"),
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
	ValidateLLDPPacket(t, msg2a, "switch1", 1024)
	t.Log("Got LLDP packet on stream switch2a")

	t.Log("Waiting for packet in on stream switch2b")
	msg2b, err := stream2b.Recv()
	assert.NoError(t, err)
	ValidateLLDPPacket(t, msg2b, "switch1", 1024)
	t.Log("Got LLDP packet on stream switch2b")

	// Prepare to clean up...
	go func() {
		time.Sleep(1 * time.Second)
		CleanUp()
	}()

	// ... and make sure we did not receive any other (unexpected) messages
	ValidateNoPendingMessage(t, stream1)
	ValidateNoPendingMessage(t, stream2a)
	ValidateNoPendingMessage(t, stream2b)
}

// ValidateLLDPPacket makes sure that the specified message is a packet in with an LLDP packet with an expected port
func ValidateLLDPPacket(t *testing.T, msg *p4api.StreamMessageResponse, chassisID string, sdnPortNumber uint32) {
	packetIn := msg.GetPacket()
	assert.NotNil(t, packetIn)
	packet := gopacket.NewPacket(packetIn.Payload, layers.LayerTypeLinkLayerDiscovery, gopacket.Default)
	assert.NotNil(t, packet)

	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	assert.NotNil(t, lldpLayer)

	lldp := lldpLayer.(*layers.LinkLayerDiscovery)
	assert.Equal(t, chassisID, string(lldp.ChassisID.ID))

	lldpPortNumber, err := strconv.ParseInt(string(lldp.PortID.ID), 10, 32)
	assert.NoError(t, err)
	assert.Equal(t, sdnPortNumber, uint32(lldpPortNumber))
}

// ValidateNoPendingMessage makes sure that the specified stream has no pending messages; blocks until message or error
func ValidateNoPendingMessage(t *testing.T, stream p4api.P4Runtime_StreamChannelClient) {
	_, err := stream.Recv()
	assert.NotNil(t, err)
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
