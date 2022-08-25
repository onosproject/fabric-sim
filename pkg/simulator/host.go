// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
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
		arp, err := utils.ARPRequestPacket(utils.IP(another.IpAddress), utils.MAC(nic.MacAddress), utils.IP(nic.IpAddress))
		if err != nil {
			log.Warnf("Host %s: Unable to serialize ARP request: %+v", hs.Host.ID, err)
			continue
		}
		deviceSim, err := hs.simulation.GetDeviceSimulatorForPort(nic.ID)
		if err != nil {
			log.Warnf("Host %s: Unable to find device simulator: %+v", hs.Host.ID, err)
			continue
		}

		deviceSim.SendPacketIn(arp, deviceSim.Ports[nic.ID].InternalNumber)
	}
}

// TODO: Additional simulation logic goes here

// SendARPResponse simulates emission of an ARP response as a packet-in on all the hosts' interfaces
func (hs *HostSimulator) SendARPResponse(another *simapi.Host) {

}

// Periodically emit ARP requests for other hosts' IP addresses
func (hs *HostSimulator) emitARPRequests() {
	for {
		select {
		case <-time.After(15 * time.Second):
			hs.emitRandomARPRequest()
		case <-hs.done:
			return
		}
	}
}

// Picks a random host (other than us) and emits an ARP query for it
func (hs *HostSimulator) emitRandomARPRequest() {
	if another := hs.simulation.GetRandomHostSimulator(); another != nil {
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
