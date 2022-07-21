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
	"sync"
)

// Simulation tracks all entities and activities related to device, host and link simulation
type Simulation struct {
	lock             sync.RWMutex
	deviceSimulators map[simapi.DeviceID]*devsim.Simulator
	linkSimulators   map[simapi.LinkID]*linksim.Simulator
	hostSimulators   map[simapi.HostID]*hostsim.Simulator
}

// NewSimulation creates a new core simulation entity
func NewSimulation() *Simulation {
	return &Simulation{
		deviceSimulators: make(map[simapi.DeviceID]*devsim.Simulator),
		linkSimulators:   make(map[simapi.LinkID]*linksim.Simulator),
		hostSimulators:   make(map[simapi.HostID]*hostsim.Simulator),
	}
}

// TODO: Rework this using generics at some point to allow same core to track different simulators

// Device inventory

// AddDeviceSimulator creates a new devices simulator for the specified device
func (i *Simulation) AddDeviceSimulator(dev *simapi.Device) (*devsim.Simulator, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	sim := devsim.NewDeviceSimulator(dev)
	if _, ok := i.deviceSimulators[dev.ID]; !ok {
		i.deviceSimulators[dev.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("Simulator already created")
}

// GetDeviceSimulators returns a list of all device simulators
func (i *Simulation) GetDeviceSimulators() []*devsim.Simulator {
	i.lock.RLock()
	defer i.lock.RUnlock()
	sims := make([]*devsim.Simulator, 0, len(i.deviceSimulators))
	for _, sim := range i.deviceSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetDeviceSimulator returns the simulator for the specified device ID
func (i *Simulation) GetDeviceSimulator(id simapi.DeviceID) (*devsim.Simulator, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	if sim, ok := i.deviceSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Simulator not found")
}

// RemoveDeviceSimulator removes the simulator for the specified device ID and stops all its related activities
func (i *Simulation) RemoveDeviceSimulator(id simapi.DeviceID) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if sim, ok := i.deviceSimulators[id]; ok {
		delete(i.deviceSimulators, id)
		sim.Stop(simapi.StopMode_ORDERLY_STOP)
		return nil
	}
	return errors.NewNotFound("Simulator not found")
}

// Link inventory

// AddLinkSimulator creates a new link simulator for the specified link
func (i *Simulation) AddLinkSimulator(link *simapi.Link) (*linksim.Simulator, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	sim := linksim.NewLinkSimulator(link)
	if _, ok := i.linkSimulators[link.ID]; !ok {
		i.linkSimulators[link.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("Simulator already created")
}

// GetLinkSimulators returns a list of all link simulators
func (i *Simulation) GetLinkSimulators() []*linksim.Simulator {
	i.lock.RLock()
	defer i.lock.RUnlock()
	sims := make([]*linksim.Simulator, 0, len(i.linkSimulators))
	for _, sim := range i.linkSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetLinkSimulator returns the simulator for the specified link ID
func (i *Simulation) GetLinkSimulator(id simapi.LinkID) (*linksim.Simulator, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	if sim, ok := i.linkSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Simulator not found")
}

// RemoveLinkSimulator removes the simulator for the specified link ID and stops all its related activities
func (i *Simulation) RemoveLinkSimulator(id simapi.LinkID) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if _, ok := i.linkSimulators[id]; ok {
		delete(i.linkSimulators, id)
		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("Simulator not found")
}

// Host inventory

// AddHostSimulator creates a new host simulator for the specified host
func (i *Simulation) AddHostSimulator(host *simapi.Host) (*hostsim.Simulator, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	sim := hostsim.NewHostSimulator(host)
	if _, ok := i.hostSimulators[host.ID]; !ok {
		i.hostSimulators[host.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("Simulator already created")
}

// GetHostSimulators returns a list of all host simulators
func (i *Simulation) GetHostSimulators() []*hostsim.Simulator {
	i.lock.RLock()
	defer i.lock.RUnlock()
	sims := make([]*hostsim.Simulator, 0, len(i.hostSimulators))
	for _, sim := range i.hostSimulators {
		sims = append(sims, sim)
	}
	return sims
}

// GetHostSimulator returns the simulator for the specified host ID
func (i *Simulation) GetHostSimulator(id simapi.HostID) (*hostsim.Simulator, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	if sim, ok := i.hostSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Simulator not found")
}

// RemoveHostSimulator removes the simulator for the specified host ID and stops all its related activities
func (i *Simulation) RemoveHostSimulator(id simapi.HostID) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if _, ok := i.hostSimulators[id]; ok {
		delete(i.hostSimulators, id)
		// TODO: Add stop as needed
		return nil
	}
	return errors.NewNotFound("Simulator not found")
}
