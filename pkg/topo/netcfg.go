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

// Netcfg structure represents ONOS network configuration
type Netcfg struct {
	Devices map[string]*NetcfgDevice `json:"devices"`
	Hosts   map[string]*NetcfgHost   `json:"hosts"`
}

// NetcfgDevice structure represents ONOS device config
type NetcfgDevice struct {
	Basic *NetcfgDeviceBasic `json:"basic"`
}

// NetcfgDeviceBasic structure represents ONOS basic device config
type NetcfgDeviceBasic struct {
	Name              string `json:"name"`
	ManagementAddress string `json:"managementAddress"`
	Driver            string `json:"driver"`
	Pipeconf          string `json:"pipeconf"`
	LocType           string `json:"locType"`
	GridX             int    `json:"gridX"`
	GridY             int    `json:"gridY"`
}

// NetcfgHost structure represents ONOS host config
type NetcfgHost struct {
	Basic *NetcfgHostBasic `json:"basic"`
}

// NetcfgHostBasic structure represents ONOS basic host config
type NetcfgHostBasic struct {
	Name    string `json:"name"`
	LocType string `json:"locType"`
	GridX   int    `json:"gridX"`
	GridY   int    `json:"gridY"`
}

// TODO: add location/position information for GUI layout

// GenerateNetcfg loads the specified topology YAML file and uses it to generate ONOS netcfg.json file
func GenerateNetcfg(topologyPath string, netcfgPath string, driver string, pipeconf string) error {
	log.Infof("Loading topology from %s", topologyPath)
	topology := &Topology{}

	if err := loadTopologyFile(topologyPath, topology); err != nil {
		return err
	}

	ncfg := &Netcfg{
		Devices: make(map[string]*NetcfgDevice),
		Hosts:   make(map[string]*NetcfgHost),
	}

	for _, device := range topology.Devices {
		ncfg.Devices[fmt.Sprintf("device:%s", device.ID)] = createNetcfgDevice(device, driver, pipeconf)
	}

	for _, host := range topology.Hosts {
		for _, nic := range host.NICs {
			ncfg.Hosts[fmt.Sprintf("%s/None", strings.ToUpper(nic.Mac))] = createNetcfgHost(host, nic)
		}
	}

	return saveNetfgFile(ncfg, netcfgPath)
}

func createNetcfgDevice(device Device, driver string, pipeconf string) *NetcfgDevice {
	loc := &GridPosition{}
	if device.Pos != nil {
		loc = device.Pos
	}
	return &NetcfgDevice{
		Basic: &NetcfgDeviceBasic{
			Name:              device.ID,
			ManagementAddress: fmt.Sprintf("grpc://fabric-sim:%d?device_id=0", device.AgentPort),
			Driver:            driver,
			Pipeconf:          pipeconf,
			LocType:           "grid",
			GridX:             loc.X,
			GridY:             loc.Y,
		},
	}
}

func createNetcfgHost(host Host, nic NIC) *NetcfgHost {
	loc := &GridPosition{}
	if host.Pos != nil {
		loc = host.Pos
	}
	return &NetcfgHost{
		Basic: &NetcfgHostBasic{
			Name:    host.ID,
			LocType: "grid",
			// TODO: Adjust this based on NIC
			GridX: loc.X,
			GridY: loc.Y,
		},
	}
}

// Saves the given netcfg as JSON in the specified file path; stdout if -
func saveNetfgFile(ncfg *Netcfg, path string) error {
	cfg := viper.New()
	cfg.Set("Devices", ncfg.Devices)
	cfg.Set("Hosts", ncfg.Hosts)

	// Create a temporary file and schedule it for removal on exit
	file, err := os.CreateTemp("", "netcfg*.json")
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
	//if _, err = fmt.Fprint(output, generatedJSONHeader); err != nil {
	//	return err
	//}

	// Then append the copy of the JSON content
	if _, err = fmt.Fprint(output, string(buffer)); err != nil {
		return err
	}
	return nil
}
