// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"math/rand"
	"sync"
	"time"
)

// HostSimulator simulates a single host
type HostSimulator struct {
	Host       *simapi.Host
	simulation *Simulation

	lock sync.RWMutex
	done chan string
}

// NewHostSimulator initializes a new device simulator
func NewHostSimulator(host *simapi.Host, simulation *Simulation) *HostSimulator {
	log.Infof("Host %s: Creating simulator", host.ID)
	return &HostSimulator{
		Host:       host,
		simulation: simulation,
		done:       make(chan string),
	}
}

// Start starts background host simulation activities, e.g. emitting ARP and DHCP packets
func (hs *HostSimulator) Start() {
	hs.lock.Lock()
	defer hs.lock.Unlock()
	go hs.emitARPRequests()
}

// Stop stops any background host simulation activities
func (hs *HostSimulator) Stop() {
	hs.done <- "stop"
}

// SendARPRequest simulates emission of an ARP request as a packet-in on all the hosts' interfaces
func (hs *HostSimulator) SendARPRequest(another *simapi.NetworkInterface) {
	for _, nic := range hs.Host.Interfaces {
		if err := hs.EmitARPRequests(nic, []string{another.IpAddress}); err != nil {
			log.Warnf("Host %s: Unable to emit ARP for %s: %v", hs.Host.ID, another.IpAddress, err)
		}
	}
}

// SendARPResponse simulates emission of an ARP response as a packet-in on all the hosts' interfaces
func (hs *HostSimulator) SendARPResponse(another *simapi.Host) {
	// TODO: implement this when needed
}

const (
	arpMinDelay = 30
	arpVardelay = 30
)

// Periodically emit ARP requests for other hosts' IP addresses
func (hs *HostSimulator) emitARPRequests() {
	for {
		select {
		case <-time.After(time.Duration(arpMinDelay+rand.Intn(arpVardelay)) * time.Second):
			hs.emitRandomARPRequest()
		case <-hs.done:
			return
		}
	}
}

// Picks a random host (other than us) and emits an ARP query for it
func (hs *HostSimulator) emitRandomARPRequest() {
	if another := hs.simulation.GetRandomHostSimulator(hs); another != nil {
		hs.SendARPRequest(another.GetRandomNetworkInterface())
	}
}

// GetRandomNetworkInterface returns randomly chosen network interface for the host
func (hs *HostSimulator) GetRandomNetworkInterface() *simapi.NetworkInterface {
	hs.lock.RLock()
	defer hs.lock.RUnlock()
	ri := rand.Intn(len(hs.Host.Interfaces))
	i := 0
	for _, nic := range hs.Host.Interfaces {
		if i == ri {
			return nic
		}
		i++
	}
	return nil
}

// EmitARPRequests triggers the specified host NIC to send ARP requests for a set of IP addresses
func (hs *HostSimulator) EmitARPRequests(nic *simapi.NetworkInterface, dstIPs []string) error {
	for _, ip := range dstIPs {
		arp, err := utils.ARPRequestPacket(utils.IP(ip), utils.MAC(nic.MacAddress), utils.IP(nic.IpAddress))
		if err != nil {
			log.Warnf("Host %s: Unable to serialize ARP request: %+v", hs.Host.ID, err)
			continue
		}
		deviceSim, err := hs.simulation.GetDeviceSimulatorForPort(nic.ID)
		if err != nil {
			log.Warnf("Host %s: Unable to find device simulator: %+v", hs.Host.ID, err)
			continue
		}
		if deviceSim.HasPuntRuleForEthType(layers.EthernetTypeARP) {
			deviceSim.SendPacketIn(arp, deviceSim.Ports[nic.ID].InternalNumber)
		}
	}
	return nil
}

// GetNetworkInterfaceByMac returns the network interface associated with the specified MAC address on this host
func (hs *HostSimulator) GetNetworkInterfaceByMac(mac string) *simapi.NetworkInterface {
	for _, nic := range hs.Host.Interfaces {
		if nic.MacAddress == mac {
			return nic
		}
	}
	return nil
}

// TODO: Additional simulation logic goes here
