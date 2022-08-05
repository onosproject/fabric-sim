// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadP4Info(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)
	t.Logf("p4info: %+v", info)
}
