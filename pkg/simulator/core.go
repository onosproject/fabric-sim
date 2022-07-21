// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package simulator contains the core simulation coordinator
package simulator

import (
	devsim "github.com/onosproject/fabric-sim/pkg/simulator/device"
	hostsim "github.com/onosproject/fabric-sim/pkg/simulator/host"
	linksim "github.com/onosproject/fabric-sim/pkg/simulator/link"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"strings"
	"sync"
)

// Simulation tracks all entities and activities related to device, host and link simulation
type Simulation struct {
	lock             sync.RWMutex
	deviceSimulators map[simapi.DeviceID]*devsim.Simulator
	linkSimulators   map[simapi.LinkID]*linksim.Simulator
	hostSimulators   map[simapi.HostID]*hostsim.Simulator

	// Auxiliary structures
	usedEgressPorts  map[simapi.PortID]*linkOrNIC
	usedIngressPorts map[simapi.PortID]*linkOrNIC
}

// NewSimulation creates a new core simulation entity
func NewSimulation() *Simulation {
	return &Simulation{
		deviceSimulators: make(map[simapi.DeviceID]*devsim.Simulator),
		linkSimulators:   make(map[simapi.LinkID]*linksim.Simulator),
		hostSimulators:   make(map[simapi.HostID]*hostsim.Simulator),
		usedEgressPorts:  make(map[simapi.PortID]*linkOrNIC),
		usedIngressPorts: make(map[simapi.PortID]*linkOrNIC),
	}
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
func (s *Simulation) AddDeviceSimulator(dev *simapi.Device) (*devsim.Simulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sim := devsim.NewDeviceSimulator(dev)
	if _, ok := s.deviceSimulators[dev.ID]; !ok {
		s.deviceSimulators[dev.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("Device %s already created", dev.ID)
}

// GetDeviceSimulators returns a list of all device simulators
func (s *Simulation) GetDeviceSimulators() []*devsim.Simulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*devsim.Simulator, 0, len(s.deviceSimulators))
	for _, sim := range s.deviceSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetDeviceSimulator returns the simulator for the specified device ID
func (s *Simulation) GetDeviceSimulator(id simapi.DeviceID) (*devsim.Simulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.deviceSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Device %s not found", id)
}

// RemoveDeviceSimulator removes the simulator for the specified device ID and stops all its related activities
func (s *Simulation) RemoveDeviceSimulator(id simapi.DeviceID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if sim, ok := s.deviceSimulators[id]; ok {
		delete(s.deviceSimulators, id)
		sim.Stop(simapi.StopMode_ORDERLY_STOP)
		return nil
	}
	return errors.NewNotFound("Device %s not found", id)
}

// Link inventory

// AddLinkSimulator creates a new link simulator for the specified link
func (s *Simulation) AddLinkSimulator(link *simapi.Link) (*linksim.Simulator, error) {
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
		return nil, errors.NewInvalid("Port %s is already used for %s", link.SrcID, lon)
	}
	if lon, ok := s.usedIngressPorts[link.TgtID]; ok {
		return nil, errors.NewInvalid("Port %s is already used for %s", link.TgtID, lon)
	}

	sim := linksim.NewLinkSimulator(link)
	if _, ok := s.linkSimulators[link.ID]; !ok {
		s.linkSimulators[link.ID] = sim
		s.usedEgressPorts[link.SrcID] = &linkOrNIC{link: link}
		s.usedIngressPorts[link.TgtID] = &linkOrNIC{link: link}
		return sim, nil
	}
	return nil, errors.NewInvalid("Link %s already created", link.ID)
}

func (s *Simulation) validatePort(id simapi.PortID) error {
	f := strings.SplitN(string(id), "/", 2)
	if len(f) < 2 {
		return errors.NewInvalid("Invalid port ID format: %s", id)
	}
	deviceID := simapi.DeviceID(f[0])
	d, ok := s.deviceSimulators[deviceID]
	if !ok {
		return errors.NewNotFound("Device %s not found", deviceID)
	}

	if _, ok := d.Ports[id]; !ok {
		return errors.NewNotFound("Port %s not found", id)
	}
	return nil
}

// GetLinkSimulators returns a list of all link simulators
func (s *Simulation) GetLinkSimulators() []*linksim.Simulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*linksim.Simulator, 0, len(s.linkSimulators))
	for _, sim := range s.linkSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetLinkSimulator returns the simulator for the specified link ID
func (s *Simulation) GetLinkSimulator(id simapi.LinkID) (*linksim.Simulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.linkSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Link %s not found", id)
}

// RemoveLinkSimulator removes the simulator for the specified link ID and stops all its related activities
func (s *Simulation) RemoveLinkSimulator(id simapi.LinkID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.linkSimulators[id]; ok {
		delete(s.linkSimulators, id)
		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("Link %s not found", id)
}

// Host inventory

// AddHostSimulator creates a new host simulator for the specified host
func (s *Simulation) AddHostSimulator(host *simapi.Host) (*hostsim.Simulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sim := hostsim.NewHostSimulator(host)

	// Validate that the port for all NICs exists
	for _, nic := range host.Interfaces {
		if err := s.validatePort(nic.ID); err != nil {
			return nil, err
		}

		// Validate that the port is in fact available
		if lon, ok := s.usedEgressPorts[nic.ID]; ok {
			return nil, errors.NewInvalid("Port %s is already used for %s", nic.ID, lon)
		}
		if lon, ok := s.usedIngressPorts[nic.ID]; ok {
			return nil, errors.NewInvalid("Port %s is already used for %s", nic.ID, lon)
		}
	}

	if _, ok := s.hostSimulators[host.ID]; !ok {
		s.hostSimulators[host.ID] = sim
		for _, nic := range host.Interfaces {
			s.usedEgressPorts[nic.ID] = &linkOrNIC{nic: nic}
			s.usedIngressPorts[nic.ID] = &linkOrNIC{nic: nic}
		}
		return sim, nil
	}
	return nil, errors.NewInvalid("Host %s already created", host.ID)
}

// GetHostSimulators returns a list of all host simulators
func (s *Simulation) GetHostSimulators() []*hostsim.Simulator {
	s.lock.RLock()
	defer s.lock.RUnlock()
	sims := make([]*hostsim.Simulator, 0, len(s.hostSimulators))
	for _, sim := range s.hostSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetHostSimulator returns the simulator for the specified host ID
func (s *Simulation) GetHostSimulator(id simapi.HostID) (*hostsim.Simulator, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if sim, ok := s.hostSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Host %s not found", id)
}

// RemoveHostSimulator removes the simulator for the specified host ID and stops all its related activities
func (s *Simulation) RemoveHostSimulator(id simapi.HostID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.hostSimulators[id]; ok {
		delete(s.hostSimulators, id)
		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("Host %s not found", id)
}
