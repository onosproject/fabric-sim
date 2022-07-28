// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var log = logging.GetLogger("topo")

// Topology is a description of a simulated network topology
type Topology struct {
	Devices []Device `mapstructure:"devices" yaml:"devices"`
	Links   []Link   `mapstructure:"links" yaml:"links"`
	Hosts   []Host   `mapstructure:"hosts" yaml:"hosts"`
}

// Device is a description of a simulated device
type Device struct {
	ID        string `mapstructure:"id" yaml:"id"`
	ChassisID uint64 `mapstructure:"chassis_id" yaml:"chassis_id"`
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

// Reads configuration from the specified path (- for stdin) via viper; ready to Unmarshal
func readConfig(path string) (*viper.Viper, error) {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	if path == "-" {
		if err := cfg.ReadConfig(os.Stdin); err != nil {
			return cfg, err
		}
	} else {
		cfg.SetConfigName(filepath.Base(path))
		cfg.AddConfigPath(filepath.Dir(path))
		if err := cfg.ReadInConfig(); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
