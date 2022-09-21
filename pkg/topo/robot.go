// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
)

// RobotTopology is a description of an expected network topology
type RobotTopology struct {
	Nodes   []*RobotNode
	Devices []*RobotDevice
	Hosts   []*RobotHost
}

// RobotNode is a description of an expected ONOS controller node
type RobotNode struct {
	IP string `mapstructure:"ip" yaml:"ip"`
}

// RobotDevice is a description of an expected device
type RobotDevice struct {
	ID    string       `mapstructure:"id" yaml:"id"`
	Links []*RobotLink `mapstructure:"links" yaml:"links"`
}

// RobotLink is a description of an expected link
type RobotLink struct {
	Target     string `mapstructure:"tgt" yaml:"tgt"`
	SourcePort string `mapstructure:"srcport" yaml:"srcport"`
	TargetPort string `mapstructure:"tgtport" yaml:"tgtport"`
}

// RobotHost is a description of an expected host
type RobotHost struct {
	ID       string           `mapstructure:"id" yaml:"id"`
	MAC      string           `mapstructure:"mac" yaml:"mac"`
	IP       string           `mapstructure:"ip" yaml:"ip"`
	Gateway  string           `mapstructure:"gw" yaml:"gw"`
	VLAN     string           `mapstructure:"vlan" yaml:"vlan"`
	TenantID string           `mapstructure:"tenantid" yaml:"tenantid"`
	Links    []*RobotHostLink `mapstructure:"links" yaml:"links"`
}

// RobotHostLink is a description of an expected host link
type RobotHostLink struct {
	Device string `mapstructure:"device" yaml:"device"`
	Port   string `mapstructure:"port" yaml:"port"`
}

// GenerateRobotTopology loads the specified topology YAML file and uses it to generate Robot topology YAML file
func GenerateRobotTopology(topologyPath string, robotTopologyPath string) error {
	log.Infof("Loading topology from %s", topologyPath)
	topology := &Topology{}

	if err := LoadTopologyFile(topologyPath, topology); err != nil {
		return err
	}

	devices := make([]*RobotDevice, 0, len(topology.Devices))
	for _, device := range topology.Devices {
		devices = append(devices, createRobotDevice(device, topology))
	}

	hosts := make([]*RobotHost, 0, len(topology.Hosts))
	for _, host := range topology.Hosts {
		for _, nic := range host.NICs {
			hosts = append(hosts, createRobotHost(host, nic, topology))
		}
	}

	rtopo := &RobotTopology{
		Nodes:   []*RobotNode{{IP: "127.0.0.1"}},
		Devices: devices,
		Hosts:   hosts,
	}
	return saveRobotTopologyFile(rtopo, robotTopologyPath)
}

func createRobotDevice(device Device, topology *Topology) *RobotDevice {
	links := make([]*RobotLink, 0)
	for _, link := range topology.Links {
		sf := strings.SplitN(link.SrcPortID, "/", 2)
		tf := strings.SplitN(link.TgtPortID, "/", 2)
		if len(sf) > 1 && len(tf) > 1 && sf[0] == device.ID {
			links = append(links, &RobotLink{Target: tf[0], SourcePort: sf[1], TargetPort: tf[1]})
		}
	}
	return &RobotDevice{ID: device.ID, Links: links}
}

func createRobotHost(host Host, nic NIC, topology *Topology) *RobotHost {
	f := strings.SplitN(nic.Port, "/", 2)
	return &RobotHost{
		ID:       host.ID,
		MAC:      nic.Mac,
		IP:       nic.IPv4,
		Gateway:  nic.IPv4,
		VLAN:     "None",
		TenantID: "0",
		Links:    []*RobotHostLink{{Device: f[0], Port: f[1]}},
	}
}

// Saves the given robot structure as YAML in the specified file path; stdout if -
func saveRobotTopologyFile(topo *RobotTopology, path string) error {
	cfg := viper.New()
	cfg.Set("ONOS_REST_PORT", 8181)
	cfg.Set("ONOS_SSH_PORT", 8101)
	cfg.Set("nodes", topo.Nodes)
	cfg.Set("devices", topo.Devices)
	cfg.Set("hosts", topo.Hosts)

	// Create a temporary file and schedule it for removal on exit
	file, err := os.CreateTemp("", "robot*.yaml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()

	// Write the configuration to the temporary file
	if err = cfg.WriteConfigAs(file.Name()); err != nil {
		return err
	}

	// Now copy the file to the intended destination; stdout if -
	buffer, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return err
	}

	output := os.Stdout
	if path != "-" {
		output, err = os.Create(path)
		if err != nil {
			return err
		}
		defer output.Close()
	}

	// Write the header comment to the path first
	if _, err = fmt.Fprint(output, generatedHeader); err != nil {
		return err
	}

	// Then append the copy of the YAML content
	if _, err = fmt.Fprint(output, string(buffer)); err != nil {
		return err
	}
	return nil
}
