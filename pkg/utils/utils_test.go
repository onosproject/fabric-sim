// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestLoadP4Info(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/p4info.txt")
	assert.NoError(t, err)
	assert.Equal(t, "tna", info.PkgInfo.Arch)

	assert.Len(t, info.Tables, 20)
	assert.Len(t, info.Actions, 40)
	assert.Len(t, info.ActionProfiles, 1)
	assert.Len(t, info.Meters, 1)
	assert.Len(t, info.Counters, 0)
	assert.Len(t, info.DirectMeters, 0)
	assert.Len(t, info.DirectCounters, 14)
	assert.Len(t, info.Digests, 0)
	assert.Len(t, info.Externs, 1)
	assert.Len(t, info.Registers, 0)
	assert.Len(t, info.ValueSets, 0)

	buf := P4InfoBytes(info)
	assert.True(t, len(buf) > 1024)

	// Test non-existent P4Info
	_, err = LoadP4Info("foobar.txt")
	assert.Error(t, err)

	// Test non-sensical P4Info
	_, err = LoadP4Info("utils_test.go")
	assert.Error(t, err)
}

func TestGeneration(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/p4info.txt")
	assert.NoError(t, err)

	tl := int32(len(info.Tables))
	for i := 0; i < 10000; i++ {
		tableInfo := info.Tables[rand.Int31n(tl)]
		GenerateTableEntry(tableInfo, 123, nil)
	}
}

func TestArbitration(t *testing.T) {
	eid := &p4api.Uint128{High: 1, Low: 2}
	mar := CreateMastershipArbitration(eid, nil)
	assert.NotNil(t, mar.GetArbitration())
	assert.Equal(t, mar.GetArbitration().ElectionId.High, uint64(1))
	assert.Equal(t, mar.GetArbitration().ElectionId.Low, uint64(2))
}
