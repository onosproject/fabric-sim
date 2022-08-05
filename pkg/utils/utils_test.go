// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestLoadP4Info(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)

	//t.Logf("p4info: %+v", info)

	assert.Equal(t, info.PkgInfo.Arch, "v1model")

	assert.Len(t, info.Tables, 22)
	assert.Len(t, info.Actions, 41)
	assert.Len(t, info.ActionProfiles, 1)
	assert.Len(t, info.Meters, 1)
	assert.Len(t, info.Counters, 4)
	assert.Len(t, info.DirectMeters, 0)
	assert.Len(t, info.DirectCounters, 15)
	assert.Len(t, info.Digests, 0)
	assert.Len(t, info.Externs, 0)
	assert.Len(t, info.Registers, 0)
	assert.Len(t, info.ValueSets, 0)
}

func TestGeneration(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)

	tl := int32(len(info.Tables))
	for i := 0; i < 10000; i++ {
		tableInfo := info.Tables[rand.Int31n(tl)]
		GenerateTableEntry(tableInfo, 123, nil)
	}
}
