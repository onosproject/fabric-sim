// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
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
	if err := loadRecipeFile(topologyPath, recipe); err != nil {
		return err
	}

	switch {
	case recipe.DevCloudFabric != nil:
		GenerateDevCloudFabric(recipe.DevCloudFabric, topologyPath)
	case recipe.AccessFabric != nil:
		GenerateAccessFabric(recipe.AccessFabric, topologyPath)
	default:
		log.Info("No supported topology recipe found")
	}
	return nil
}

// Loads the specified topology recipe YAML file
func loadRecipeFile(path string, recipe *Recipe) error {
	if err := readConfig(path); err != nil {
		return err
	}
	return viper.Unmarshal(recipe)
}
