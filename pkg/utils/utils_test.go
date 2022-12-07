// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestGeneration(t *testing.T) {
	info, err := p4utils.LoadP4Info("../../pipelines/p4info.txt")
	assert.NoError(t, err)

	tl := int32(len(info.Tables))
	for i := 0; i < 10000; i++ {
		tableInfo := info.Tables[rand.Int31n(tl)]
		GenerateTableEntry(tableInfo, 123, nil)
	}
}

func TestArbitration(t *testing.T) {
	eid := &p4api.Uint128{High: 1, Low: 2}
	mar := p4utils.CreateMastershipArbitration(eid, nil)
	assert.NotNil(t, mar.GetArbitration())
	assert.Equal(t, mar.GetArbitration().ElectionId.High, uint64(1))
	assert.Equal(t, mar.GetArbitration().ElectionId.Low, uint64(2))
}
