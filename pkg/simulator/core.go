// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package simulator contains the core simulation coordinator
package simulator

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-api/go/onos/stratum"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"math/rand"
	"strings"
	"sync"
)

var log = logging.GetLogger("simulator")

// Simulation tracks all entities and activities related to device, host and link simulation
type Simulation struct {
	lock             sync.RWMutex
	deviceSimulators map[simapi.DeviceID]*DeviceSimulator
	linkSimulators   map[simapi.LinkID]*LinkSimulator
	hostSimulators   map[simapi.HostID]*HostSimulator
	Collector        *StatsCollector

	// Auxiliary structures
	usedEgressPorts  map[simapi.PortID]*linkOrNIC
	usedIngressPorts map[simapi.PortID]*linkOrNIC
}

// NewSimulation creates a new core simulation entity
func NewSimulation() *Simulation {
	simulation := &Simulation{
		deviceSimulators: make(map[simapi.DeviceID]*DeviceSimulator),
		linkSimulators:   make(map[simapi.LinkID]*LinkSimulator),
		hostSimulators:   make(map[simapi.HostID]*HostSimulator),
		usedEgressPorts:  make(map[simapi.PortID]*linkOrNIC),
		usedIngressPorts: make(map[simapi.PortID]*linkOrNIC),
	}
	simulation.Collector = newStatsCollector(simulation)
	return simulation
}

// DeviceAgent is an abstraction of P4Runtime and gNMI NB server
type DeviceAgent interface {
	// Start starts the simulated device agent
	Start(simulation *Simulation, deviceSim *DeviceSimulator) error

	// Stop stops the simulated device agent
	Stop(mode simapi.StopMode) error
}

// ConnectionTracker is an abstraction of an entity maintaining a peer connection
type ConnectionTracker interface {
	// GetConnection returns the peer connection information
	GetConnection() *simapi.Connection
}

// StreamResponder is an abstraction for sending StreamResponse messages to controllers
type StreamResponder interface {
	ConnectionTracker

	// LatchMastershipArbitration record the mastership arbitration role and election ID if the arbitration update is not nil
	LatchMastershipArbitration(arbitration *p4api.MasterArbitrationUpdate) *p4api.MasterArbitrationUpdate

	// SendMastershipArbitration sends a mastership arbitration message to the controller with OK code if
	// the controller has the master election ID or with the given fail code otherwise
	SendMastershipArbitration(role *p4api.Role, masterElectionID *p4api.Uint128, failCode code.Code)

	// Send queues up the specified response to asynchronously sends on the backing stream
	Send(response *p4api.StreamMessageResponse)

	// IsMaster returns true if the responder is the current master, i.e. has the master election ID, for the given role.
	IsMaster(role *p4api.Role, masterElectionID *p4api.Uint128) bool

	// GetRoleConfig returns the stratum role configuration received during role arbitration
	GetRoleConfig() *stratum.P4RoleConfig
}

// SubscribeResponder is an abstraction for sending SubscribeResponse messages to controllers
type SubscribeResponder interface {
	ConnectionTracker

	// Send queues up the specified response to asynchronously sends on the backing stream
	Send(response *gnmi.SubscribeResponse)
}

type linkOrNIC struct {
	link *simapi.Link
	nic  *simapi.NetworkInterface
}

func (l *linkOrNIC) String() string {
	if l.nic != nil {
		return l.nic.MacAddress
	}
	return string(l.link.ID)
}

// TODO: Rework this using generics at some point to allow same core to track different simulators

// Device inventory

// AddDeviceSimulator creates a new devices simulator for the specified device
func (s *Simulation) AddDeviceSimulator(dev *simapi.Device, agent DeviceAgent) (*DeviceSimulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sim := NewDeviceSimulator(dev, agent, s)
	if _, ok := s.deviceSimulators[dev.ID]; !ok {
		s.deviceSimulators[dev.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("device %s already created", dev.ID)
}

// GetDeviceSimulators returns a list of all device simulators
func (s *Simulation) GetDeviceSimulators() []*DeviceSimulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*DeviceSimulator, 0, len(s.deviceSimulators))
	for _, sim := range s.deviceSimulators {
		sims = append(sims, sim.SnapshotStats())
	}
	return sims
}

// GetDeviceSimulator returns the simulator for the specified device ID
func (s *Simulation) GetDeviceSimulator(id simapi.DeviceID) (*DeviceSimulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.deviceSimulators[id]; ok {
		return sim.SnapshotStats(), nil
	}
	return nil, errors.NewNotFound("device %s not found", id)
}

// GetDeviceSimulatorForPort returns the simulator for the specified device port ID
func (s *Simulation) GetDeviceSimulatorForPort(id simapi.PortID) (*DeviceSimulator, error) {
	deviceID, err := ExtractDeviceID(id)
	if err != nil {
		return nil, err
	}
	return s.GetDeviceSimulator(deviceID)
}

// RemoveDeviceSimulator removes the simulator for the specified device ID and stops all its related activities
func (s *Simulation) RemoveDeviceSimulator(id simapi.DeviceID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if sim, ok := s.deviceSimulators[id]; ok {
		sim.Stop(simapi.StopMode_CHAOTIC_STOP)
		delete(s.deviceSimulators, id)
		return nil
	}
	return errors.NewNotFound("device %s not found", id)
}

// Link inventory

// AddLinkSimulator creates a new link simulator for the specified link
func (s *Simulation) AddLinkSimulator(link *simapi.Link) (*LinkSimulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Validate that the source and target ports exist
	if err := s.validatePort(link.SrcID); err != nil {
		return nil, err
	}
	if err := s.validatePort(link.TgtID); err != nil {
		return nil, err
	}

	// Validate that the port is in fact available
	if lon, ok := s.usedEgressPorts[link.SrcID]; ok {
		log.Errorf("Port %s is already used for %s", link.SrcID, lon)
		return nil, errors.NewInvalid("port %s is already used for %s", link.SrcID, lon)
	}
	if lon, ok := s.usedIngressPorts[link.TgtID]; ok {
		log.Errorf("Port %s is already used for %s", link.TgtID, lon)
		return nil, errors.NewInvalid("port %s is already used for %s", link.TgtID, lon)
	}

	sim := NewLinkSimulator(link)
	if _, ok := s.linkSimulators[link.ID]; !ok {
		s.linkSimulators[link.ID] = sim
		s.usedEgressPorts[link.SrcID] = &linkOrNIC{link: link}
		s.usedIngressPorts[link.TgtID] = &linkOrNIC{link: link}
		return sim, nil
	}
	return nil, errors.NewInvalid("link %s already created", link.ID)
}

func (s *Simulation) validatePort(id simapi.PortID) error {
	deviceID, err := ExtractDeviceID(id)
	if err != nil {
		return err
	}
	d, ok := s.deviceSimulators[deviceID]
	if !ok {
		log.Errorf("Device %s not found for port %s", deviceID, id)
		return errors.NewNotFound("device %s not found", deviceID)
	}

	if _, ok = d.Ports[id]; !ok {
		log.Errorf("Port %s not found for device %s", id, deviceID)
		return errors.NewNotFound("port %s not found", id)
	}
	return nil
}

// ExtractDeviceID extracts the device ID from the port ID
func ExtractDeviceID(id simapi.PortID) (simapi.DeviceID, error) {
	f := strings.SplitN(string(id), "/", 2)
	if len(f) < 2 {
		return "", errors.NewInvalid("invalid port ID format: %s", id)
	}
	deviceID := simapi.DeviceID(f[0])
	return deviceID, nil
}

// GetLinkSimulators returns a list of all link simulators
func (s *Simulation) GetLinkSimulators() []*LinkSimulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*LinkSimulator, 0, len(s.linkSimulators))
	for _, sim := range s.linkSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetLinkSimulator returns the simulator for the specified link ID
func (s *Simulation) GetLinkSimulator(id simapi.LinkID) (*LinkSimulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.linkSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("link %s not found", id)
}

// RemoveLinkSimulator removes the simulator for the specified link ID and stops all its related activities
func (s *Simulation) RemoveLinkSimulator(id simapi.LinkID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if sim, ok := s.linkSimulators[id]; ok {
		delete(s.linkSimulators, id)
		delete(s.usedEgressPorts, sim.Link.SrcID)
		delete(s.usedIngressPorts, sim.Link.TgtID)
		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("link %s not found", id)
}

// Host inventory

// AddHostSimulator creates a new host simulator for the specified host
func (s *Simulation) AddHostSimulator(host *simapi.Host) (*HostSimulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sim := NewHostSimulator(host, s)

	// Validate that the port for all NICs exists
	for _, nic := range host.Interfaces {
		if err := s.validatePort(nic.ID); err != nil {
			return nil, err
		}

		// Validate that the port is in fact available
		if lon, ok := s.usedEgressPorts[nic.ID]; ok {
			return nil, errors.NewInvalid("port %s is already used for %s", nic.ID, lon)
		}
		if lon, ok := s.usedIngressPorts[nic.ID]; ok {
			return nil, errors.NewInvalid("port %s is already used for %s", nic.ID, lon)
		}
	}

	if _, ok := s.hostSimulators[host.ID]; !ok {
		s.hostSimulators[host.ID] = sim
		for _, nic := range host.Interfaces {
			s.usedEgressPorts[nic.ID] = &linkOrNIC{nic: nic}
			s.usedIngressPorts[nic.ID] = &linkOrNIC{nic: nic}
		}
		sim.Start() // start the host simulator
		return sim, nil
	}
	return nil, errors.NewInvalid("host %s already created", host.ID)
}

// GetHostSimulators returns a list of all host simulators
func (s *Simulation) GetHostSimulators() []*HostSimulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*HostSimulator, 0, len(s.hostSimulators))
	for _, sim := range s.hostSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetHostSimulator returns the simulator for the specified host ID
func (s *Simulation) GetHostSimulator(id simapi.HostID) (*HostSimulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.hostSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("host %s not found", id)
}

// GetRandomHostSimulator returns a random host simulator; except the specified one, if not nil
func (s *Simulation) GetRandomHostSimulator(except *HostSimulator) *HostSimulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if len(s.hostSimulators) > 0 {
		ri := rand.Intn(len(s.hostSimulators))
		i := 0
		for _, hs := range s.hostSimulators {
			if i == ri {
				if hs == except && len(s.hostSimulators) > 1 {
					// If we landed on the exception and there are at least two hosts, try our luck again
					return s.GetRandomHostSimulator(except)
				} else if hs == except {
					// If we landed on the exception and there is at most one host, return nil
					return nil
				}
				return hs
			}
			i++
		}
	}
	return nil
}

// RemoveHostSimulator removes the simulator for the specified host ID and stops all its related activities
func (s *Simulation) RemoveHostSimulator(id simapi.HostID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if sim, ok := s.hostSimulators[id]; ok {
		sim.Stop()
		delete(s.hostSimulators, id)
		for _, nic := range sim.Host.Interfaces {
			delete(s.usedEgressPorts, nic.ID)
			delete(s.usedIngressPorts, nic.ID)
		}

		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("host %s not found", id)
}

// GetLinkFromPort returns the link that originates from the specified device port; nil if none
func (s *Simulation) GetLinkFromPort(portID simapi.PortID) *simapi.Link {
	if ln, ok := s.usedEgressPorts[portID]; ok {
		return ln.link // if the port is used for a NIC, this will be nil, which is what we want
	}
	return nil
}

// EmitARPs triggers the specified host NIC to send ARP requests for a set of IP addresses
func (s *Simulation) EmitARPs(id simapi.HostID, mac string, ips []string) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sim, ok := s.hostSimulators[id]
	if !ok {
		return errors.NewNotFound("host %s not found", id)
	}
	if nic := sim.GetNetworkInterfaceByMac(mac); nic != nil {
		return sim.EmitARPRequests(nic, ips)
	}
	return errors.NewNotFound("nic with MAC %s not found", mac)
}
