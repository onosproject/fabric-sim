// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package simulator contains the core simulation coordinator
package simulator

import (
	"github.com/onosproject/fabric-sim/pkg/simulator/device"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sync"
)

type Simulation struct {
	lock             sync.RWMutex
	deviceSimulators map[simapi.DeviceID]*device.DeviceSimulator

	// HostSimulators
	// LinkSimulators
}

func NewSimulation() *Simulation {
	return &Simulation{
		deviceSimulators: make(map[simapi.DeviceID]*device.DeviceSimulator),
	}
}

// TODO: Rework this using generics at some point to allow same core to track different simulators

func (i *Simulation) AddDeviceSimulator(dev *simapi.Device) (*device.DeviceSimulator, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	sim := device.NewDeviceSimulator(dev)
	if _, ok := i.deviceSimulators[dev.ID]; !ok {
		i.deviceSimulators[dev.ID] = sim
		return sim, nil
	}
	return nil, errors.NewInvalid("Simulator already created")
}

func (i *Simulation) GetDeviceSimulators() []*device.DeviceSimulator {
	i.lock.RLock()
	defer i.lock.RUnlock()
	sims := make([]*device.DeviceSimulator, len(i.deviceSimulators))
	for _, sim := range i.deviceSimulators {
		sims = append(sims, sim)
	}
	return sims
}

func (i *Simulation) GetDeviceSimulator(id simapi.DeviceID) (*device.DeviceSimulator, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	if sim, ok := i.deviceSimulators[id]; ok {
		return sim, nil
	}
	return nil, errors.NewNotFound("Simulator not found")
}

func (i *Simulation) RemoveDeviceSimulator(id simapi.DeviceID) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if _, ok := i.deviceSimulators[id]; ok {
		delete(i.deviceSimulators, id)
		return nil
	}
	return errors.NewNotFound("Simulator not found")
}
