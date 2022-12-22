// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package main is the main entry point for starting fabric simulator
package main

import (
	"github.com/onosproject/fabric-sim/pkg/manager"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/spf13/cobra"
)

var log = logging.GetLogger()

// The main entry point
func main() {
	cmd := &cobra.Command{
		Use:  "fabric-sim",
		RunE: runRootCommand,
	}
	cli.AddServiceEndpointFlags(cmd, "fabric-sim gRPC")
	cli.Run(cmd)
}

func runRootCommand(cmd *cobra.Command, args []string) error {
	flags, err := cli.ExtractServiceEndpointFlags(cmd)
	if err != nil {
		return err
	}

	log.Info("Starting fabric-sim")
	return cli.RunDaemon(manager.NewManager(manager.Config{ServiceFlags: flags}))
}
