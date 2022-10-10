// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/fabric-sim/pkg/topo"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"os"
)

const (
	serviceAddress = "fabric-sim:5150"

	topologyFlag = "topology"
	recipeFlag   = "recipe"
	outputFlag   = "output"
	driverFlag   = "driver"
	pipeconfFlag = "pipeconf"
	tenants      = "tenants"
)

// The main entry point
func main() {
	if err := getRootCommand().Execute(); err != nil {
		println(err)
		os.Exit(1)
	}
}

func getRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fabric-sim-topo {load, clear, generate}",
		Short: "Load, clear or generate simulated topology",
	}
	cmd.AddCommand(getLoadCommand())
	cmd.AddCommand(getClearCommand())
	cmd.AddCommand(getGenerateCommand())
	return cmd
}

func getLoadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "load",
		Aliases: []string{"start"},
		Short:   "Load fabric topology from a YAML file and start the simulation",
		Args:    cobra.NoArgs,
		RunE:    runLoadCommand,
	}
	cli.AddEndpointFlags(cmd, serviceAddress)
	cmd.Flags().String(topologyFlag, "-", "topology YAML file; use - for stdin (default)")
	return cmd
}

func runLoadCommand(cmd *cobra.Command, args []string) error {
	conn, err := cli.GetConnection(cmd)
	if err != nil {
		return err
	}
	defer closeConnection(conn)
	topologyPath, _ := cmd.Flags().GetString(topologyFlag)
	return topo.LoadTopology(conn, topologyPath)
}

func getClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clear",
		Aliases: []string{"stop"},
		Short:   "Stop the simulation and clear the entire simulated fabric topology",
		Args:    cobra.NoArgs,
		RunE:    runClearCommand,
	}
	cli.AddEndpointFlags(cmd, serviceAddress)
	return cmd
}

func runClearCommand(cmd *cobra.Command, args []string) error {
	conn, err := cli.GetConnection(cmd)
	if err != nil {
		return err
	}
	defer closeConnection(conn)
	return topo.ClearTopology(conn)
}

func closeConnection(conn *grpc.ClientConn) {
	_ = conn.Close()
}

func getGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate {topology, netcfg, robot}",
		Aliases: []string{"gen"},
		Short:   "Generate fabric topology YAML, netcfg JSON, or Robot tests topology YAML",
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(getGenerateTopoCommand())
	cmd.AddCommand(getGenerateNetcfgCommand())
	cmd.AddCommand(getGenerateRobotCommand())
	return cmd
}

func getGenerateTopoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "topology",
		Aliases: []string{"topo"},
		Short:   "Generate a simulated fabric topology from a topology recipe YAML file",
		Args:    cobra.NoArgs,
		RunE:    runGenerateTopoCommand,
	}
	cmd.Flags().String(recipeFlag, "-", "topology recipe YAML file; use - for stdin (default)")
	cmd.Flags().String(outputFlag, "-", "output topology YAML file; use - for stdout (default)")
	return cmd
}

func runGenerateTopoCommand(cmd *cobra.Command, args []string) error {
	recipePath, _ := cmd.Flags().GetString(recipeFlag)
	outputPath, _ := cmd.Flags().GetString(outputFlag)
	return topo.GenerateTopology(recipePath, outputPath)
}

func getGenerateNetcfgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netcfg",
		Short: "Generate netcfg JSON file from the specified topology YAML file",
		Args:  cobra.NoArgs,
		RunE:  runGenerateNetcfgCommand,
	}
	cmd.Flags().String(topologyFlag, "-", "topology YAML file; use - for stdin (default)")
	cmd.Flags().String(driverFlag, "stratum-tofino", "ONOS driver")
	cmd.Flags().String(pipeconfFlag, "org.stratumproject.fabric.montara_sde_9_7_0", "ONOS pipeconf")
	cmd.Flags().String(outputFlag, "-", "netcfg JSON file; use - for stdout (default)")
	cmd.Flags().IntSlice(tenants, []int{1, 1, 2, 3, 3, 4, 4, 2, 5, 3, 5, 6, 6, 2, 2, 7, 7},
		"pattern list of tenants for assigning logical ports to")
	return cmd
}

func runGenerateNetcfgCommand(cmd *cobra.Command, args []string) error {
	topologyPath, _ := cmd.Flags().GetString(topologyFlag)
	outputPath, _ := cmd.Flags().GetString(outputFlag)
	driver, _ := cmd.Flags().GetString(driverFlag)
	pipeconf, _ := cmd.Flags().GetString(pipeconfFlag)
	tenants, _ := cmd.Flags().GetIntSlice(tenants)
	return topo.GenerateNetcfg(topologyPath, outputPath, driver, pipeconf, tenants)
}

func getGenerateRobotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "robot",
		Short: "Generate Robot topology YAML file from the specified topology YAML file",
		Args:  cobra.NoArgs,
		RunE:  runGenerateRobotCommand,
	}
	cmd.Flags().String(topologyFlag, "-", "topology YAML file; use - for stdin (default)")
	cmd.Flags().String(outputFlag, "-", "Robot topology YAML file; use - for stdout (default)")
	return cmd
}

func runGenerateRobotCommand(cmd *cobra.Command, args []string) error {
	topologyPath, _ := cmd.Flags().GetString(topologyFlag)
	outputPath, _ := cmd.Flags().GetString(outputFlag)
	return topo.GenerateRobotTopology(topologyPath, outputPath)
}
