// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"encoding/binary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/utils"
	"github.com/onosproject/fabric-sim/test/client"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/code"
	"testing"
	"time"
)

var codec *utils.ControllerMetadataCodec

// TestLLDPPacket tests the LLDP packet-out handling
func (s *TestSuite) TestLLDPPacket(t *testing.T) {
	LoadAndValidate(t, "topologies/trivial.yaml", 2, 2, 2,
		func(*simapi.Device) int { return 2 }, func(*simapi.Host) int { return 1 })
	defer CleanUp(t)

	// Let's create a codec for meta-data from the P4 info file
	info, err := utils.LoadP4Info("pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)
	codec = utils.NewControllerMetadataCodec(info)

	conn, err := client.CreateConnection()
	assert.NoError(t, err)
	defer conn.Close()

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

	eID1 := p4api.Uint128{High: 0, Low: 2}
	eID2 := p4api.Uint128{High: 0, Low: 1}

	stream1 := StartStream(ctx, t, p4sw1, &eID1, int32(code.Code_OK))
	stream2a := StartStream(ctx, t, p4sw2a, &eID1, int32(code.Code_OK))
	stream2b := StartStream(ctx, t, p4sw2b, &eID2, int32(code.Code_ALREADY_EXISTS))

	err = ApplyPipelineConfig(ctx, p4sw1, r1.Device.ChassisID, "", &eID1, 321, info)
	assert.NoError(t, err)
	err = ApplyPipelineConfig(ctx, p4sw2a, r2.Device.ChassisID, "", &eID1, 321, info)
	assert.NoError(t, err)

	// Install an entry to punt LLDP packets to CPU
	err = InstallPuntRule(ctx, p4sw2a, r2.Device.ChassisID, "", &eID1, uint16(layers.EthernetTypeLinkLayerDiscovery))
	assert.NoError(t, err)

	egressPort := uint32(224)
	lldpBytes, err := utils.ControllerLLDPPacket(string(r1.Device.ID), egressPort)
	assert.NoError(t, err)

	err = stream1.Send(&p4api.StreamMessageRequest{
		Update: &p4api.StreamMessageRequest_Packet{
			Packet: &p4api.PacketOut{
				Payload:  lldpBytes,
				Metadata: codec.EncodePacketOutMetadata(&utils.PacketOutMetadata{EgressPort: egressPort}),
			}},
	})
	assert.NoError(t, err)

	t.Log("Waiting for packet in on stream switch2a")
	msg2a, err := stream2a.Recv()
	assert.NoError(t, err)
	ValidateLLDPPacket(t, msg2a, "switch1", 234, 224)
	t.Log("Got LLDP packet on stream switch2a")

	t.Log("Waiting for packet in on stream switch2b")
	msg2b, err := stream2b.Recv()
	assert.NoError(t, err)
	ValidateLLDPPacket(t, msg2b, "switch1", 234, 224)
	t.Log("Got LLDP packet on stream switch2b")

	// Prepare to clean up...
	go func() {
		time.Sleep(1 * time.Second)
		CleanUp(t)
	}()

	// ... and make sure we did not receive any other (unexpected) messages
	ValidateNoPendingMessage(t, stream1)
	ValidateNoPendingMessage(t, stream2a)
	ValidateNoPendingMessage(t, stream2b)
}

// InstallPuntRule installs rule matching on the specified eth type with action to punt to CPU
func InstallPuntRule(ctx context.Context, p4sw2a p4api.P4RuntimeClient, chassisID uint64, roleName string, electionID *p4api.Uint128, ethType uint16) error {
	mask := []byte{0xff, 0xff}
	ethTypeValue := []byte{0, 0}
	binary.BigEndian.PutUint16(ethTypeValue, ethType)

	_, err := p4sw2a.Write(ctx, &p4api.WriteRequest{
		DeviceId:   chassisID,
		Role:       roleName,
		ElectionId: electionID,
		Updates: []*p4api.Update{{
			Type: p4api.Update_INSERT,
			Entity: &p4api.Entity{Entity: &p4api.Entity_TableEntry{
				TableEntry: &p4api.TableEntry{
					TableId: 44104738,
					Match: []*p4api.FieldMatch{{
						FieldId: 5,
						FieldMatchType: &p4api.FieldMatch_Ternary_{
							Ternary: &p4api.FieldMatch_Ternary{
								Value: ethTypeValue,
								Mask:  mask,
							},
						},
					}},
					Action: &p4api.TableAction{
						Type: &p4api.TableAction_Action{
							Action: &p4api.Action{
								ActionId: 23579892,
							},
						},
					},
				}}},
		}},
	})
	return err
}

// ValidateLLDPPacket makes sure that the specified message is a packet in with an LLDP packet with an expected port
func ValidateLLDPPacket(t *testing.T, msg *p4api.StreamMessageResponse, chassisID string, ingressPortNumber uint32, sdnPortNumber uint32) {
	packetIn := msg.GetPacket()
	assert.NotNil(t, packetIn)

	if packetIn != nil {
		pim := codec.DecodePacketInMetadata(packetIn.Metadata)
		assert.NotNil(t, pim)
		assert.Equal(t, ingressPortNumber, pim.IngressPort)

		packet := gopacket.NewPacket(packetIn.Payload, layers.LayerTypeEthernet, gopacket.Default)
		assert.NotNil(t, packet)

		lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
		assert.NotNil(t, lldpLayer)

		//lldp := lldpLayer.(*layers.LinkLayerDiscovery)
		//assert.Equal(t, chassisID, string(lldp.ChassisID.ID))
	}
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

	err = stream.Send(utils.CreateMastershipArbitration(electionID, nil))
	assert.NoError(t, err)

	msg, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, electionCode, msg.GetArbitration().Status.Code)

	return stream
}
