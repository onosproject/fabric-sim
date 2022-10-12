// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const genPortConfig = false

// Netcfg structure represents ONOS network configuration
type Netcfg struct {
	Devices map[string]*NetcfgDevice `json:"devices"`
	Ports   map[string]*NetcfgPort   `json:"ports"`
	Hosts   map[string]*NetcfgHost   `json:"hosts"`
}

// NetcfgDevice structure represents ONOS device config
type NetcfgDevice struct {
	Basic          *NetcfgDeviceBasic          `json:"basic"`
	Underlay       *NetcfgDeviceUnderlay       `json:"underlay"`
	Reconciliation *NetcfgDeviceReconciliation `json:"reconciliation"`
}

// NetcfgDeviceBasic structure represents ONOS basic device config
type NetcfgDeviceBasic struct {
	Name                         string                     `json:"name"`
	ManagementAddress            string                     `json:"managementAddress"`
	AncillaryManagementAddresses *NetcfgManagementAddresses `json:"ancillaryManagementAddresses,omitempty"`
	Driver                       string                     `json:"driver"`
	Pipeconf                     string                     `json:"pipeconf"`
	LocType                      string                     `json:"locType"`
	GridX                        int                        `json:"gridX"`
	GridY                        int                        `json:"gridY"`
}

// NetcfgManagementAddresses holds local agent addresses
type NetcfgManagementAddresses struct {
	HostLocalAgent string `json:"host-local-agent"`
}

// NetcfgDeviceUnderlay holds underlay config
type NetcfgDeviceUnderlay struct {
	NodeSid      int      `json:"nodeSid"`
	Loopbacks    []string `json:"loopbacks"`
	RouterMac    string   `json:"routerMac"`
	IsEdgeRouter bool     `json:"isEdgeRouter"`
}

// NetcfgDeviceReconciliation holds reconciliation config
type NetcfgDeviceReconciliation struct {
	RequiredApps []string `json:"requiredApps"`
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

// NetcfgPort structure holds port configuration
type NetcfgPort struct {
	Interfaces []*NetcfgPortInterfaces `json:"interfaces"`
}

// NetcfgPortInterfaces represents a single port interface configuration
type NetcfgPortInterfaces struct {
	Name         string   `json:"name"`
	Ips          []string `json:"ips"`
	VlanUntagged int      `json:"vlan-untagged,omitempty"`
	VlanTagged   []int    `json:"vlan-tagged,omitempty"`
	Mac          string   `json:"mac"`
}

// TODO: add location/position information for GUI layout

// GenerateNetcfg loads the specified topology YAML file and uses it to generate ONOS netcfg.json file
func GenerateNetcfg(topologyPath string, netcfgPath string, driver string, pipeconf string, tenants []int) error {
	log.Infof("Loading topology from %s", topologyPath)
	topology := &Topology{}

	if err := LoadTopologyFile(topologyPath, topology); err != nil {
		return err
	}

	ncfg := &Netcfg{
		Devices: make(map[string]*NetcfgDevice),
		Ports:   make(map[string]*NetcfgPort),
		Hosts:   make(map[string]*NetcfgHost),
	}

	portMap := make(map[string]string)

	for _, device := range topology.Devices {
		ncfg.Devices[fmt.Sprintf("device:%s", device.ID)] = createNetcfgDevice(device, driver, pipeconf)
		for _, port := range device.Ports {
			portMap[fmt.Sprintf("%s/%d", device.ID, port.Number)] =
				fmt.Sprintf("%s/%d", device.ID, port.SDNNumber)
		}
	}

	onosCommands := make([]string, 0)
	ti := 0
	for _, host := range topology.Hosts {
		for _, nic := range host.NICs {
			if genPortConfig {
				portID := fmt.Sprintf("device:%s", nic.Port)
				ncfg.Ports[portID] = createNetcfgPort(portID, host, nic)
			}
			onosHostID := fmt.Sprintf("%s/None[%d]", strings.ToUpper(nic.Mac), tenants[ti])
			ncfg.Hosts[onosHostID] = createNetcfgHost(host, nic)
			onosCommands = append(onosCommands, createONOSCommand(nic, tenants[ti], portMap))
			ti = (ti + 1) % len(tenants)
		}
	}

	_ = saveONOSCommands(netcfgPath, onosCommands)
	return saveNetcfgFile(ncfg, netcfgPath)
}

func createONOSCommand(nic NIC, tenant int, portMap map[string]string) string {
	port := portMap[nic.Port]
	f := indexPattern.FindAllString(port, 2)
	deviceIndex, _ := strconv.ParseUint(f[0], 10, 32)
	portIndex, _ := strconv.ParseUint(f[1], 10, 32)

	return fmt.Sprintf("create-logical-switch-port t%dl%dp%d %d device:%s %d false\n",
		tenant, deviceIndex, portIndex, tenant, port, tenant*10)
}

func createNetcfgDevice(device Device, driver string, pipeconf string) *NetcfgDevice {
	loc := &GridPosition{}
	if device.Pos != nil {
		loc = device.Pos
	}

	index := getIndex(device.ID)
	underlay := &NetcfgDeviceUnderlay{}
	reconciliation := &NetcfgDeviceReconciliation{RequiredApps: []string{"org.onosproject.underlay"}}

	var ancillary *NetcfgManagementAddresses
	useDriver := driver
	usePipeconf := pipeconf
	if isLeaf(device.ID) {
		leafIndex := getIndex(device.ID)
		ancillary = &NetcfgManagementAddresses{
			HostLocalAgent: fmt.Sprintf("grpc://sdfabric-switch-host-agent-%d.sdfabric-switch-host-agent:11161", leafIndex-1),
		}
		useDriver = fmt.Sprintf("%s-la", driver)
		usePipeconf = strings.Replace(usePipeconf, ".fabric.", ".fabric-vn.", 1)
		underlay.NodeSid = 100 + index
		underlay.Loopbacks = []string{fmt.Sprintf("192.168.1.%d", index)}
		underlay.RouterMac = fmt.Sprintf("00:AA:00:00:00:%02d", index)
		underlay.IsEdgeRouter = true
		reconciliation.RequiredApps = append(reconciliation.RequiredApps, "org.onosproject.virtualnetworking")
		reconciliation.RequiredApps = append(reconciliation.RequiredApps, "org.onosproject.localagents")
	} else {
		underlay.NodeSid = 200 + index
		underlay.Loopbacks = []string{fmt.Sprintf("192.168.2.%d", index)}
		underlay.RouterMac = fmt.Sprintf("00:BB:00:00:00:%02d", index)
	}

	return &NetcfgDevice{
		Basic: &NetcfgDeviceBasic{
			Name:                         device.ID,
			ManagementAddress:            fmt.Sprintf("grpc://fabric-sim:%d?device_id=0", device.AgentPort),
			AncillaryManagementAddresses: ancillary,
			Driver:                       useDriver,
			Pipeconf:                     usePipeconf,
			LocType:                      "grid",
			GridX:                        loc.X,
			GridY:                        loc.Y,
		},
		Underlay:       underlay,
		Reconciliation: reconciliation,
	}
}

func isLeaf(id string) bool {
	return strings.HasPrefix(id, "leaf")
}

var indexPattern = regexp.MustCompile("([0-9]+)")

func getIndex(id string) int {
	match := indexPattern.FindStringIndex(id)
	if len(match) < 2 || match[1] == 0 {
		return 1
	}
	index, err := strconv.ParseUint(id[match[0]:match[1]], 10, 16)
	if err != nil {
		return 0
	}
	return int(index)
}

func createNetcfgPort(portID string, host Host, nic NIC) *NetcfgPort {
	name := strings.ReplaceAll(strings.ReplaceAll(portID, "/", "-"), "[", "")
	return &NetcfgPort{Interfaces: []*NetcfgPortInterfaces{{
		Name:         strings.Replace(name, "device:", "", 1),
		Ips:          []string{fmt.Sprintf("%s/24", nic.IPv4)},
		VlanUntagged: 100,
		VlanTagged:   nil,
		Mac:          fmt.Sprintf("00:AA:00:00:00:%02d", getIndex(portID)),
	}}}
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
func saveNetcfgFile(ncfg *Netcfg, path string) error {
	cfg := viper.New()
	cfg.Set("devices", ncfg.Devices)
	cfg.Set("ports", ncfg.Ports)
	cfg.Set("hosts", ncfg.Hosts)

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

func saveONOSCommands(path string, commands []string) error {
	f, err := os.Create(strings.Replace(strings.Replace(path, ".json", ".cmds", 1), "_netcfg", "", 1))
	if err != nil {
		return err
	}
	defer f.Close()

	for _, cmd := range commands {
		_, _ = f.WriteString(cmd)
	}
	return nil
}
