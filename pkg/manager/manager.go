// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package manager contains the simulator manager coordinating lifecycle of the NB API and simulation controller
package manager

import (
	simapi "github.com/onosproject/fabric-sim/pkg/northbound/fabricsim"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	ServiceFlags *cli.ServiceEndpointFlags
}

// Manager is single point of entry for the fabric-sim
type Manager struct {
	cli.Daemon
	Config     Config
	Simulation *simulator.Simulation
}

// NewManager initializes the application manager
func NewManager(cfg Config) *Manager {
	log.Infow("Creating manager")
	return &Manager{Config: cfg}
}

// Start initializes and starts the core simulator and the NB gRPC API.
func (m *Manager) Start() error {
	log.Info("Starting Manager")

	// Initialize the simulation core
	m.Simulation = simulator.NewSimulation()
	m.Simulation.Collector.Start()

	// Starts NB server
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}
	return nil
}

// Stop stops the manager
func (m *Manager) Stop() {
	log.Info("Stopping Manager")
}

// startSouthboundServer starts the northbound gRPC server
func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(cli.ServerConfigFromFlags(m.Config.ServiceFlags, northbound.SecurityConfig{}))
	s.AddService(logging.Service{})
	s.AddService(simapi.NewService(m.Simulation))

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}
