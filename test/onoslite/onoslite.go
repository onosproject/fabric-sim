// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package onoslite contains implementation of a ultra-light controller that simulates ONOS interactions with
// the network environment
package onoslite

import (
	"context"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/utils"
	"github.com/onosproject/fabric-sim/test/basic"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	gnoiapi "github.com/openconfig/gnoi/system"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math/rand"
	"sync"
	"time"
)

var log = logging.GetLogger("onoslite")

/*
	* Device discovery
		- load tuples of (chassisID, agent port) to prime discovery
		- P4RT connection establishment
			- mastership arbitration
			- pipeline reconciliation/configuration
			- flow rule installation for ARP, LLDP, etc. punt to CPU
		- gNMI-based port discovery
		- gNOI-based device liveness (System.GetTime)
	* Link discovery
		- periodic emission of LLDP packet-out
		- intercept of LLDP packet-in
	* Host discovery
		- intercept of ARP packet-in

	* Go API to access the NIB (devices, links, hosts)
		- for validation of topology using simulator recipe
*/

// LiteONOS is an ultra-light controller to test fabric-sim
type LiteONOS struct {
	DevicePointers []*DevicePointer
	Devices        map[string]*Device
	Links          map[string]*Link
	Hosts          map[string]*Host

	lock sync.RWMutex
}

// DevicePointer is a structure holding information required to prime device discovery
type DevicePointer struct {
	ID          string
	ChassisID   uint64
	ControlPort int32
}

// Device is a simple representation of a device discovered and controlled by the ONOS lite
type Device struct {
	ID      string
	Pointer *DevicePointer
	Ports   map[string]*Port

	conn       *grpc.ClientConn
	p4Client   p4api.P4RuntimeClient
	gnmiClient gnmi.GNMIClient
	gnoiClient gnoiapi.SystemClient

	cookie     uint64
	electionID *p4api.Uint128
	codec      *utils.ControllerMetadataCodec
	stream     p4api.P4Runtime_StreamChannelClient
	ctx        context.Context
	ctxCancel  context.CancelFunc
}

// Port is a simple representation of a device port discovered by the ONOS lite
type Port struct {
	ID     string
	Number uint32
	Status string
}

// Link is a simple representation of a link discovered by the ONOS lite
type Link struct {
	ID        string
	SrcPortID string
	TgtPortID string
}

// Host is a simple representation of a host network interface discovered by the ONOS lite
type Host struct {
	MAC string
	IP  string
}

// NewLiteONOS creates a new ONOS lite object
func NewLiteONOS() *LiteONOS {
	return &LiteONOS{
		DevicePointers: nil,
		Devices:        make(map[string]*Device),
		Links:          make(map[string]*Link),
		Hosts:          make(map[string]*Host),
	}
}

// Start starts the controller and primes its device discovery with the specified list of device pointers
func (o *LiteONOS) Start(pointers []*DevicePointer) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	if len(o.DevicePointers) > 0 {
		return errors.NewInvalid("already started")
	}
	o.DevicePointers = pointers
	for _, dp := range o.DevicePointers {
		go o.discoverDevice(dp)
	}
	return nil
}

// Stop stops the controller and any of its background processes
func (o *LiteONOS) Stop() error {
	o.lock.Lock()
	defer o.lock.Unlock()
	if len(o.DevicePointers) == 0 {
		return errors.NewInvalid("not started")
	}
	o.DevicePointers = nil
	for _, device := range o.Devices {
		device.ctxCancel()
	}
	return nil
}

func (o *LiteONOS) addDevice(device *Device) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if _, ok := o.Devices[device.ID]; !ok {
		o.Devices[device.ID] = device
	}
}

func (o *LiteONOS) addLink(srcPort string, tgtPort string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	linkID := fmt.Sprintf("%s-%s", srcPort, tgtPort)
	if _, ok := o.Links[linkID]; !ok {
		o.Links[linkID] = &Link{
			ID:        linkID,
			SrcPortID: srcPort,
			TgtPortID: tgtPort,
		}
	}
}

func (o *LiteONOS) addHost(macString string, ipString string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if _, ok := o.Hosts[macString]; !ok {
		o.Hosts[macString] = &Host{
			MAC: macString,
			IP:  ipString,
		}
	}
}

func (o *LiteONOS) discoverDevice(dp *DevicePointer) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	device := &Device{
		ID:        dp.ID,
		Pointer:   dp,
		cookie:    rand.Uint64(),
		ctx:       ctx,
		ctxCancel: ctxCancel,
	}
	var err error
	for device.ctx != nil {
		if err = o.establishDeviceConnection(ctx, device); err == nil {
			o.addDevice(device)
			if err = o.reconcilePipelineConfig(ctx, device); err == nil {
				if err = o.installFlowRules(ctx, device); err == nil {
					// Add iteration around this...
					if err = o.discoverPorts(ctx, device); err == nil {
						if err = o.discoverLinks(device); err == nil {
							time.Sleep(1 * time.Minute)
						}
					}
				}
			}
		}

		if err != nil {
			log.Warnf("%s: %+v", device.ID, err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (o *LiteONOS) establishDeviceConnection(ctx context.Context, device *Device) error {
	log.Infof("%s: connecting...", device.ID)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	var err error
	device.conn, err = grpc.Dial(fmt.Sprintf("fabric-sim:%d", device.Pointer.ControlPort), opts...)
	if err != nil {
		return err
	}

	device.p4Client = p4api.NewP4RuntimeClient(device.conn)
	device.gnmiClient = gnmi.NewGNMIClient(device.conn)
	device.gnoiClient = gnoiapi.NewSystemClient(device.conn)

	// Establish stream and issue mastership
	if device.stream, err = device.p4Client.StreamChannel(ctx); err != nil {
		return err
	}

	device.electionID = &p4api.Uint128{Low: 123, High: 0}
	if err = device.stream.Send(utils.CreateMastershipArbitration(device.electionID)); err != nil {
		return err
	}

	var msg *p4api.StreamMessageResponse
	if msg, err = device.stream.Recv(); err != nil {
		return err
	}
	mar := msg.GetArbitration()
	if mar == nil {
		return errors.NewInvalid("%s: did not receive mastership arbitration", device.ID)
	}
	if mar.ElectionId == nil || mar.ElectionId.High != device.electionID.High || mar.ElectionId.Low != device.electionID.Low {
		return errors.NewInvalid("%s: did not win election", device.ID)
	}
	go o.monitorStream(device)
	return nil
}

func (o *LiteONOS) monitorStream(device *Device) {
	log.Infof("%s: monitoring message stream", device.ID)
	for {
		msg, err := device.stream.Recv()
		if err != nil {
			log.Warnf("%s: unable to read stream response: %+v", device.ID, err)
			return
		}

		if msg.GetPacket() != nil {
			if err := o.processPacket(msg.GetPacket(), device); err != nil {
				log.Warnf("%s: unable to process packet-in: %+v", device.ID, err)
			}
		}
	}
}

func (o *LiteONOS) processPacket(packetIn *p4api.PacketIn, device *Device) error {
	packet := gopacket.NewPacket(packetIn.Payload, layers.LayerTypeEthernet, gopacket.Default)
	pim := device.codec.DecodePacketInMetadata(packetIn.Metadata)

	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	if lldpLayer != nil {
		lldp := lldpLayer.(*layers.LinkLayerDiscovery)
		o.addLink(fmt.Sprintf("%s/%s", string(lldp.PortID.ID), string(lldp.ChassisID.ID)),
			fmt.Sprintf("%s/%d", device.ID, pim.IngressPort))
	}

	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		o.addHost(utils.MACString(arp.SourceHwAddress), utils.IPString(arp.SourceProtAddress))
	}
	return nil
}

func (o *LiteONOS) reconcilePipelineConfig(ctx context.Context, device *Device) error {
	log.Infof("%s: configuring pipeline...", device.ID)
	// ask for the pipeline config cookie
	gr, err := device.p4Client.GetForwardingPipelineConfig(ctx, &p4api.GetForwardingPipelineConfigRequest{
		DeviceId:     device.Pointer.ChassisID,
		ResponseType: p4api.GetForwardingPipelineConfigRequest_COOKIE_ONLY,
	})
	if err != nil {
		return err
	}

	// if that matches our cookie, we're good
	if device.cookie == gr.Config.Cookie.Cookie {
		return nil
	}

	// otherwise load pipeline config
	info, err := utils.LoadP4Info("pipelines/fabric-spgw-int.p4info.txt")
	if err != nil {
		return err
	}

	device.codec = utils.NewControllerMetadataCodec(info)

	// and then apply it to the device
	_, err = device.p4Client.SetForwardingPipelineConfig(ctx, &p4api.SetForwardingPipelineConfigRequest{
		DeviceId:   device.Pointer.ChassisID,
		Role:       "",
		ElectionId: device.electionID,
		Action:     p4api.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config: &p4api.ForwardingPipelineConfig{
			P4Info:         info,
			P4DeviceConfig: []byte{0, 1, 2, 3},
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: device.cookie},
		},
	})
	return err
}

func (o *LiteONOS) installFlowRules(ctx context.Context, device *Device) error {
	if err := basic.InstallPuntRule(ctx, device.p4Client, device.Pointer.ChassisID, device.electionID, uint16(layers.EthernetTypeLinkLayerDiscovery)); err != nil {
		return err
	}
	if err := basic.InstallPuntRule(ctx, device.p4Client, device.Pointer.ChassisID, device.electionID, uint16(layers.EthernetTypeARP)); err != nil {
		return err
	}
	return nil
}

func (o *LiteONOS) discoverPorts(ctx context.Context, device *Device) error {
	resp, err := device.gnmiClient.Get(ctx, &gnmi.GetRequest{
		Path: []*gnmi.Path{utils.ToPath("interfaces/interface[name=...]/state")},
	})
	if err != nil {
		return err
	}
	if len(resp.Notification) == 0 {
		return errors.NewInvalid("%s: no port data received", device.ID)
	}
	device.Ports = make(map[string]*Port)
	for _, update := range resp.Notification[0].Update {
		port := getPort(device, update.Path.Elem[1].Key["name"])
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

func (o *LiteONOS) discoverLinks(device *Device) error {
	for _, port := range device.Ports {
		lldpBytes, err := utils.ControllerLLDPPacket(device.ID, port.Number)
		if err != nil {
			return err
		}

		err = device.stream.Send(&p4api.StreamMessageRequest{
			Update: &p4api.StreamMessageRequest_Packet{
				Packet: &p4api.PacketOut{
					Payload:  lldpBytes,
					Metadata: device.codec.EncodePacketOutMetadata(&utils.PacketOutMetadata{EgressPort: port.Number}),
				}},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getPort(device *Device, id string) *Port {
	port, ok := device.Ports[id]
	if !ok {
		port = &Port{ID: id}
		device.Ports[id] = port
	}
	return port
}
