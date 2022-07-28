// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/spf13/viper"
)

// Recipe is a container for holding one of the supported simulated topology recipes
type Recipe struct {
	DevCloudFabric *DevCloudFabric `mapstructure:"dev_cloud_fabric" yaml:"dev_cloud_fabric"`
	AccessFabric   *AccessFabric   `mapstructure:"access_fabric" yaml:"access_fabric"`
	// Add more recipes here
}

// DevCloudFabric is a recipe for creating simulated dev-cloud fabric with superspines
type DevCloudFabric struct {
	SuperSpines    int `mapstructure:"super_spines" yaml:"super_spines"`
	RackPairs      int `mapstructure:"rack_pairs" yaml:"rack_pairs"`
	SpinesPerRack  int `mapstructure:"spines_per_rack" yaml:"spines_per_rack"`
	LeavesPerSpine int `mapstructure:"leaves_per_spine" yaml:"leaves_per_spine"`
	NodesPerRack   int `mapstructure:"nodes_per_rack" yaml:"nodes_per_rack"`
}

// AccessFabric is a recipe for creating simulated access fabric
type AccessFabric struct {
	Spines         int `mapstructure:"super_spines" yaml:"super_spines"`
	LeavesPerSpine int `mapstructure:"leaves_per_spine" yaml:"leaves_per_spine"`
	HostsPerLeaf   int `mapstructure:"hosts_per_leaf" yaml:"hosts_per_leaf"`
}

// GenerateTopology loads the specified topology recipe YAML file and uses the recipe to
// generate a fully elaborated topology YAML file that can be loaded via LoadTopology
func GenerateTopology(recipePath string, topologyPath string) error {
	log.Infof("Loading topology recipe from %s", recipePath)
	recipe := &Recipe{}
	if err := loadRecipeFile(recipePath, recipe); err != nil {
		return err
	}

	var topology *Topology
	switch {
	case recipe.DevCloudFabric != nil:
		topology = GenerateDevCloudFabric(recipe.DevCloudFabric)
	case recipe.AccessFabric != nil:
		topology = GenerateAccessFabric(recipe.AccessFabric)
	default:
		return errors.NewInvalid("No supported topology recipe found")
	}
	return saveTopologyFile(topology, topologyPath)
}

// Loads the specified topology recipe YAML file
func loadRecipeFile(path string, recipe *Recipe) error {
	cfg, err := readConfig(path)
	if err != nil {
		return err
	}
	return cfg.Unmarshal(recipe)
}

// Saves the given topology as YAML in the specified file path; stdout if -
func saveTopologyFile(topology *Topology, path string) error {
	cfg := viper.New()
	cfg.Set("topology", topology)
	// TODO: Implement writing to stdout
	return cfg.WriteConfigAs(path)
}

// Returns count or the default count if the count is 0
func defaultCount(count int, defaultCount int) int {
	if count > 0 {
		return count
	}
	return defaultCount
}
