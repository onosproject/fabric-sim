// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package onoslite

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-test/pkg/onostest"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the basic test suite
type TestSuite struct {
	testSuite
}

const fabricSimComponentName = "fabric-sim"
const cliComponentName = "onos-cli"

// SetupTestSuite sets up the fabric simulator basic test suite
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	registry := c.GetArg("registry").String("")
	err := helm.Chart(fabricSimComponentName, onostest.OnosChartRepo).
		Release(fabricSimComponentName).
		Set("image.tag", "latest").
		Set("global.image.registry", registry).
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart(cliComponentName, onostest.OnosChartRepo).
		Release(cliComponentName).
		Set("image.tag", "latest").
		Set("global.image.registry", registry).
		Install(true)
	if err != nil {
		return err
	}
	return nil
}
