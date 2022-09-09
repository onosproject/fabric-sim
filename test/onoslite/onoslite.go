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

	cookie         uint64
	electionID     *p4api.Uint128
	codec          *utils.ControllerMetadataCodec
	stream         p4api.P4Runtime_StreamChannelClient
	lastUpdateTime uint64
	ctx            context.Context
	ctxCancel      context.CancelFunc
	halted         bool
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
		// Stagger the starts a bit for added adversity
		time.Sleep(time.Duration(1000+rand.Intn(3000)) * time.Millisecond)

		device := newDevice(dp)
		go device.startControl(o)
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
		device.stopControl()
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
	if err = d.stream.Send(utils.CreateMastershipArbitration(d.electionID)); err != nil {
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
		onos.addLink(fmt.Sprintf("%s/%s", string(lldp.PortID.ID), string(lldp.ChassisID.ID)),
			fmt.Sprintf("%s/%d", d.ID, pim.IngressPort))
	}

	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		onos.addHost(utils.MACString(arp.SourceHwAddress), utils.IPString(arp.SourceProtAddress))
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
	info, err := utils.LoadP4Info("pipelines/fabric-spgw-int.p4info.txt")
	if err != nil {
		return err
	}

	d.codec = utils.NewControllerMetadataCodec(info)

	// and then apply it to the device
	_, err = d.p4Client.SetForwardingPipelineConfig(d.ctx, &p4api.SetForwardingPipelineConfigRequest{
		DeviceId:   d.Pointer.ChassisID,
		Role:       "",
		ElectionId: d.electionID,
		Action:     p4api.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config: &p4api.ForwardingPipelineConfig{
			P4Info:         info,
			P4DeviceConfig: []byte{0, 1, 2, 3},
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: d.cookie},
		},
	})
	return err
}

func (d *Device) installFlowRules() error {
	if err := basic.InstallPuntRule(d.ctx, d.p4Client, d.Pointer.ChassisID, d.electionID, uint16(layers.EthernetTypeLinkLayerDiscovery)); err != nil {
		return err
	}
	if err := basic.InstallPuntRule(d.ctx, d.p4Client, d.Pointer.ChassisID, d.electionID, uint16(layers.EthernetTypeARP)); err != nil {
		return err
	}
	return nil
}

func (d *Device) discoverPortsAndLinks() error {
	log.Infof("%s: (re)discovering links and ports...", d.ID)
	if err := d.discoverPorts(); err != nil {
		return err
	}
	return d.discoverLinks()
}

func (d *Device) discoverPorts() error {
	resp, err := d.gnmiClient.Get(d.ctx, &gnmi.GetRequest{
		Path: []*gnmi.Path{utils.ToPath("interfaces/interface[name=...]/state")},
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
		lldpBytes, err := utils.ControllerLLDPPacket(d.ID, port.Number)
		if err != nil {
			return err
		}

		err = d.stream.Send(&p4api.StreamMessageRequest{
			Update: &p4api.StreamMessageRequest_Packet{
				Packet: &p4api.PacketOut{
					Payload:  lldpBytes,
					Metadata: d.codec.EncodePacketOutMetadata(&utils.PacketOutMetadata{EgressPort: port.Number}),
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
