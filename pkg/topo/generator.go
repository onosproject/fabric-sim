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
	SuperSpineFabric *SuperSpineFabric `mapstructure:"superspine_fabric" yaml:"superspine_fabric"`
	AccessFabric     *AccessFabric     `mapstructure:"access_fabric" yaml:"access_fabric"`
	// Add more recipes here
}

// SuperSpineFabric is a recipe for creating simulated 4 rack fabric with superspines
type SuperSpineFabric struct {
	// Add any parametrization here, if needed
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
	case recipe.SuperSpineFabric != nil:
		topology = GenerateSuperSpineFabric(recipe.SuperSpineFabric)
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
