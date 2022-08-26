// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

// Netcfg structure represents ONOS network configuration
type Netcfg struct {
	Devices map[string]*NetcfgDevice `json:"devices"`
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
	}

	for _, device := range topology.Devices {
		ncfg.Devices[fmt.Sprintf("device:%s", device.ID)] = createNetcfgDevice(device, driver, pipeconf)
	}

	return saveNetfgFile(ncfg, netcfgPath)
}

func createNetcfgDevice(device Device, driver string, pipeconf string) *NetcfgDevice {
	return &NetcfgDevice{
		Basic: &NetcfgDeviceBasic{
			Name:              device.ID,
			ManagementAddress: fmt.Sprintf("http://fabric-sim:%d?device_id=0", device.AgentPort),
			Driver:            driver,
			Pipeconf:          pipeconf,
		},
	}
}

// Saves the given netcfg as JSON in the specified file path; stdout if -
func saveNetfgFile(ncfg *Netcfg, path string) error {
	cfg := viper.New()
	cfg.Set("Devices", ncfg.Devices)

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
