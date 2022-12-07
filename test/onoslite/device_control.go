// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package onoslite contains implementation of a ultra-light controller that simulates ONOS interactions with
// the network environment
package onoslite

import (
	"context"
	"fmt"
	gogo "github.com/gogo/protobuf/types"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/test/basic"
	"github.com/onosproject/onos-api/go/onos/stratum"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-net-lib/pkg/gnmiutils"
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
	packets "github.com/onosproject/onos-net-lib/pkg/packet"
	"github.com/openconfig/gnmi/proto/gnmi"
	gnoiapi "github.com/openconfig/gnoi/system"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"math/rand"
	"time"
)

const onosRoleName = "onos"
const deviceFlowCount = 8192

var role = newONOSRole()

func newONOSRole() *p4api.Role {
	roleConfig := &stratum.P4RoleConfig{
		PacketInFilter: &stratum.P4RoleConfig_PacketFilter{
			MetadataId: 4,
			Value:      []byte("\x01"),
		},
		ReceivesPacketIns: true,
		CanPushPipeline:   true,
	}
	any, _ := gogo.MarshalAny(roleConfig)
	return &p4api.Role{
		Name: onosRoleName,
		Config: &anypb.Any{
			TypeUrl: any.TypeUrl,
			Value:   any.Value,
		},
	}
}

func newDevice(dp *DevicePointer) *Device {
	ctx, ctxCancel := context.WithCancel(context.Background())
	device := &Device{
		ID:        dp.ID,
		Pointer:   dp,
		cookie:    rand.Uint64(),
		ctx:       ctx,
		ctxCancel: ctxCancel,
	}
	return device
}

func (d *Device) startControl(onos *LiteONOS) {
	var err error
	for !d.halted {
		if err = d.establishDeviceConnection(); err == nil {
			onos.addDevice(d)
			go d.monitorStream(onos)
			if err = d.reconcilePipelineConfig(); err == nil {
				err = d.installFlowRules()
				for !d.halted && err == nil {
					if err = d.discoverPortsAndLinks(); err == nil {
						if err = d.testLiveness(); err == nil {
							d.pause(5 * time.Second)
						}
					}
				}
			}
		}

		if err != nil {
			log.Warnf("%s: %+v", d.ID, err)
		}
		d.pause(10 * time.Second)
	}
}

func (d *Device) stopControl() {
	d.halted = true
	d.ctxCancel()
}

func (d *Device) pause(duration time.Duration) {
	select {
	case <-d.ctx.Done():
	case <-time.After(duration):
	}
}

func (d *Device) establishDeviceConnection() error {
	log.Infof("%s: connecting...", d.ID)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	var err error
	d.conn, err = grpc.Dial(fmt.Sprintf("fabric-sim:%d", d.Pointer.ControlPort), opts...)
	if err != nil {
		return err
	}

	d.p4Client = p4api.NewP4RuntimeClient(d.conn)
	d.gnmiClient = gnmi.NewGNMIClient(d.conn)
	d.gnoiClient = gnoiapi.NewSystemClient(d.conn)

	// Establish stream and issue mastership
	if d.stream, err = d.p4Client.StreamChannel(d.ctx); err != nil {
		return err
	}

	d.electionID = &p4api.Uint128{Low: 123, High: 0}
	if err = d.stream.Send(p4utils.CreateMastershipArbitration(d.electionID, role)); err != nil {
		return err
	}

	var msg *p4api.StreamMessageResponse
	if msg, err = d.stream.Recv(); err != nil {
		return err
	}
	mar := msg.GetArbitration()
	if mar == nil {
		return errors.NewInvalid("%s: did not receive mastership arbitration", d.ID)
	}
	if mar.ElectionId == nil || mar.ElectionId.High != d.electionID.High || mar.ElectionId.Low != d.electionID.Low {
		return errors.NewInvalid("%s: did not win election", d.ID)
	}
	return nil
}

func (d *Device) monitorStream(onos *LiteONOS) {
	log.Infof("%s: monitoring message stream", d.ID)
	for {
		msg, err := d.stream.Recv()
		if err != nil {
			log.Warnf("%s: unable to read stream response: %+v", d.ID, err)
			return
		}

		if msg.GetPacket() != nil {
			if err := d.processPacket(msg.GetPacket(), onos); err != nil {
				log.Warnf("%s: unable to process packet-in: %+v", d.ID, err)
			}
		}
	}
}

func (d *Device) processPacket(packetIn *p4api.PacketIn, onos *LiteONOS) error {
	packet := gopacket.NewPacket(packetIn.Payload, layers.LayerTypeEthernet, gopacket.Default)
	pim := d.codec.DecodePacketInMetadata(packetIn.Metadata)

	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	if lldpLayer != nil {
		lldp := lldpLayer.(*layers.LinkLayerDiscovery)
		onos.addLink(fmt.Sprintf("%s/%s", string(lldp.ChassisID.ID), string(lldp.PortID.ID)),
			fmt.Sprintf("%s/%d", d.ID, pim.IngressPort))
	}

	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		onos.addHost(packets.MACString(arp.SourceHwAddress), packets.IPString(arp.SourceProtAddress),
			fmt.Sprintf("%s/%d", d.ID, pim.IngressPort))
	}
	return nil
}

func (d *Device) reconcilePipelineConfig() error {
	log.Infof("%s: configuring pipeline...", d.ID)
	// ask for the pipeline config cookie
	gr, err := d.p4Client.GetForwardingPipelineConfig(d.ctx, &p4api.GetForwardingPipelineConfigRequest{
		DeviceId:     d.Pointer.ChassisID,
		ResponseType: p4api.GetForwardingPipelineConfigRequest_COOKIE_ONLY,
	})
	if err != nil {
		return err
	}

	// if that matches our cookie, we're good
	if d.cookie == gr.Config.Cookie.Cookie {
		return nil
	}

	// otherwise load pipeline config
	if d.info, err = p4utils.LoadP4Info("pipelines/p4info.txt"); err != nil {
		return err
	}

	d.codec = p4utils.NewControllerMetadataCodec(d.info)

	// and then apply it to the device
	_, err = d.p4Client.SetForwardingPipelineConfig(d.ctx, &p4api.SetForwardingPipelineConfigRequest{
		DeviceId:   d.Pointer.ChassisID,
		Role:       onosRoleName,
		ElectionId: d.electionID,
		Action:     p4api.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config: &p4api.ForwardingPipelineConfig{
			P4Info:         d.info,
			P4DeviceConfig: []byte{0, 1, 2, 3},
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: d.cookie},
		},
	})
	return err
}

func (d *Device) installFlowRules() error {
	if err := basic.InstallPuntRule(d.ctx, d.p4Client, d.Pointer.ChassisID, onosRoleName, d.electionID, uint16(layers.EthernetTypeLinkLayerDiscovery)); err != nil {
		return err
	}
	if err := basic.InstallPuntRule(d.ctx, d.p4Client, d.Pointer.ChassisID, onosRoleName, d.electionID, uint16(layers.EthernetTypeARP)); err != nil {
		return err
	}
	if err := d.installScaleFlows(d.ctx, d.p4Client, d.Pointer.ChassisID, onosRoleName, d.electionID); err != nil {
		return err
	}
	return nil
}

func (d *Device) installScaleFlows(ctx context.Context, client p4api.P4RuntimeClient, chassisID uint64, roleName string, electionID *p4api.Uint128) error {
	writeRequest := &p4api.WriteRequest{
		DeviceId:   chassisID,
		Role:       roleName,
		ElectionId: electionID,
		Atomicity:  p4api.WriteRequest_CONTINUE_ON_ERROR,
	}
	return basic.GenerateAndWriteTableEntries(ctx, client, writeRequest, d.info, deviceFlowCount)
}

func (d *Device) discoverPortsAndLinks() error {
	log.Debugf("%s: (re)discovering links and ports...", d.ID)
	if err := d.discoverPorts(); err != nil {
		return err
	}
	return d.discoverLinks()
}

func (d *Device) discoverPorts() error {
	resp, err := d.gnmiClient.Get(d.ctx, &gnmi.GetRequest{
		Path: []*gnmi.Path{gnmiutils.ToPath("interfaces/interface[name=...]/state")},
	})
	if err != nil {
		return err
	}
	if len(resp.Notification) == 0 {
		return errors.NewInvalid("%s: no port data received", d.ID)
	}
	d.Ports = make(map[string]*Port)
	for _, update := range resp.Notification[0].Update {
		port := d.getPort(update.Path.Elem[1].Key["name"])
		last := len(update.Path.Elem) - 1
		switch update.Path.Elem[last].Name {
		case "id":
			port.Number = uint32(update.Val.GetUintVal())
		case "oper-status":
			port.Status = update.Val.GetStringVal()
		}
	}
	return nil
}

func (d *Device) getPort(id string) *Port {
	port, ok := d.Ports[id]
	if !ok {
		port = &Port{ID: id}
		d.Ports[id] = port
	}
	return port
}

func (d *Device) discoverLinks() error {
	for _, port := range d.Ports {
		lldpBytes, err := packets.ControllerLLDPPacket(d.ID, port.Number)
		if err != nil {
			return err
		}

		err = d.stream.Send(&p4api.StreamMessageRequest{
			Update: &p4api.StreamMessageRequest_Packet{
				Packet: &p4api.PacketOut{
					Payload:  lldpBytes,
					Metadata: d.codec.EncodePacketOutMetadata(&p4utils.PacketOutMetadata{EgressPort: port.Number}),
				}},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Device) testLiveness() error {
	log.Debugf("%s: testing liveness", d.ID)
	resp, err := d.gnoiClient.Time(d.ctx, &gnoiapi.TimeRequest{})
	if err != nil {
		return err
	}
	d.lastUpdateTime = resp.Time
	return nil
}
