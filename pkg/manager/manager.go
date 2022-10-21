// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	simapi "github.com/onosproject/fabric-sim/pkg/northbound/fabricsim"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath   string
	KeyPath  string
	CertPath string
	GRPCPort int
	NoTLS    bool
}

// Manager single point of entry for the fabric-sim
type Manager struct {
	Config     Config
	Simulation *simulator.Simulation
}

// NewManager initializes the application manager
func NewManager(cfg Config) *Manager {
	log.Infow("Creating manager")
	mgr := Manager{
		Config: cfg,
	}
	return &mgr
}

// Run runs manager
func (m *Manager) Run() {
	log.Infow("Starting Manager")

	if err := m.Start(); err != nil {
		log.Fatalw("Unable to run Manager", "error", err)
	}
}

// Start initializes and starts the core simulator and the NB gRPC API.
func (m *Manager) Start() error {
	// Initialize the simulation core
	m.Simulation = simulator.NewSimulation()

	// Starts NB server
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}
	return nil
}

// startSouthboundServer starts the northbound gRPC server
func (m *Manager) startNorthboundServer() error {
	cfg := northbound.NewInsecureServerConfig(int16(m.Config.GRPCPort))
	if !m.Config.NoTLS {
		northbound.NewServerCfg(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, int16(m.Config.GRPCPort),
			true, northbound.SecurityConfig{})
	}
	s := northbound.NewServer(cfg)
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

// Close kills the manager
func (m *Manager) Close() {
	log.Infow("Closing Manager")
}
