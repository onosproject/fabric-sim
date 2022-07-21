// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"github.com/spf13/viper"
	"path/filepath"
)

// Topology is a description of a simulated network topology
type Topology struct {
	Devices []Device `mapstructure:"devices" yaml:"devices"`
	Links   []Link   `mapstructure:"links" yaml:"links"`
	Hosts   []Host   `mapstructure:"hosts" yaml:"hosts"`
}

// Device is a description of a simulated device
type Device struct {
	ID        string `mapstructure:"id" yaml:"id"`
	Type      string `mapstructure:"type" yaml:"type"`
	AgentPort int32  `mapstructure:"agent_port" yaml:"agent_port"`
	Stopped   bool   `mapstructure:"stopped" yaml:"stopped"`
	Ports     []Port `mapstructure:"ports" yaml:"ports"`
	// TODO: add others
}

// Port is a description of a simulated port
type Port struct {
	Number    uint32 `mapstructure:"number" yaml:"number"`
	SDNNumber uint32 `mapstructure:"sdn_number" yaml:"sdn_number"`
	Speed     string `mapstructure:"speed" yaml:"speed"`
	// TODO: add others
}

// Link is a description of a simulated link
type Link struct {
	SrcPortID      string `mapstructure:"src" yaml:"src"`
	TgtPortID      string `mapstructure:"tgt" yaml:"tgt"`
	Unidirectional bool   `mapstructure:"unidirectional" yaml:"unidirectional"`
	// TODO: add others
}

// Host is a description of a simulated host
type Host struct {
	ID   string `mapstructure:"id" yaml:"id"`
	NICs []NIC  `mapstructure:"nics" yaml:"nics"`
	// TODO: add others
}

// NIC is a description of a simulated NIC
type NIC struct {
	Mac  string `mapstructure:"mac" yaml:"mac"`
	IPv4 string `mapstructure:"ip" yaml:"ip"`
	IPV6 string `mapstructure:"ipv6" yaml:"ipv6"`
	Port string `mapstructure:"port" yaml:"port"`
	// TODO: add others
}

// LoadTopologyFile loads the specified topology YAML file
func LoadTopologyFile(path string, topology *Topology) error {
	viper.SetConfigType("yaml")
	viper.SetConfigName(filepath.Base(path))
	viper.AddConfigPath(filepath.Dir(path))

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(topology)
}
