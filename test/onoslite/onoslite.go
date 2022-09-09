// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package onoslite contains implementation of a ultra-light controller that simulates ONOS interactions with
// the network environment
package onoslite

import (
	"context"
	"fmt"
	"github.com/onosproject/fabric-sim/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	gnoiapi "github.com/openconfig/gnoi/system"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
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
