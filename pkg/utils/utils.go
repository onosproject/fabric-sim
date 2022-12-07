// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package utils contains various utilities for testing fabric simulator
package utils

import (
	"crypto/rand"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"math"
)

// GenerateTableEntry generates a table entry compliant with the specified table schema
func GenerateTableEntry(tableInfo *p4info.Table, priority int32, action *p4api.TableAction) *p4api.TableEntry {
	tableAction := action
	if action == nil {
		tableAction = GenerateTableAction(tableInfo)
	}
	entry := &p4api.TableEntry{
		TableId:          tableInfo.Preamble.Id,
		Match:            GenerateFieldMatches(tableInfo),
		Action:           tableAction,
		Priority:         priority,
		MeterConfig:      nil,
		CounterData:      nil,
		MeterCounterData: nil,
		IsDefaultAction:  false,
		IdleTimeoutNs:    0,
		TimeSinceLastHit: nil,
		Metadata:         nil,
	}
	return entry
}

// GenerateTableAction generates a table action compliant with the specified table schema
func GenerateTableAction(tableInfo *p4info.Table) *p4api.TableAction {
	action := &p4api.TableAction{
		Type: &p4api.TableAction_Action{Action: &p4api.Action{
			ActionId: tableInfo.ActionRefs[0].Id,
		}},
	}
	return action
}

// GenerateFieldMatches generates field matches compliant with the specified table schema
func GenerateFieldMatches(tableInfo *p4info.Table) []*p4api.FieldMatch {
	matches := make([]*p4api.FieldMatch, 0)
	for _, mf := range tableInfo.MatchFields {
		matches = append(matches, GenerateFieldMatch(mf))
	}
	return matches
}

// GenerateFieldMatch generates field match compliant with the specified match schema
func GenerateFieldMatch(mf *p4info.MatchField) *p4api.FieldMatch {
	matchType := mf.GetMatchType()
	match := &p4api.FieldMatch{FieldId: mf.Id}
	switch {
	case matchType == p4info.MatchField_EXACT:
		match.FieldMatchType = &p4api.FieldMatch_Exact_{Exact: &p4api.FieldMatch_Exact{
			Value: RandomBytes(mf.Bitwidth),
		}}
	case matchType == p4info.MatchField_LPM:
		match.FieldMatchType = &p4api.FieldMatch_Lpm{Lpm: &p4api.FieldMatch_LPM{
			PrefixLen: int32(RandomBytes(8)[0]),
			Value:     RandomBytes(mf.Bitwidth),
		}}
	case matchType == p4info.MatchField_TERNARY:
		match.FieldMatchType = &p4api.FieldMatch_Ternary_{Ternary: &p4api.FieldMatch_Ternary{
			Mask:  RandomBytes(mf.Bitwidth),
			Value: RandomBytes(mf.Bitwidth),
		}}
	case matchType == p4info.MatchField_RANGE:
		match.FieldMatchType = &p4api.FieldMatch_Range_{Range: &p4api.FieldMatch_Range{
			Low:  RandomBytes(mf.Bitwidth),
			High: RandomBytes(mf.Bitwidth),
		}}
	case matchType == p4info.MatchField_OPTIONAL:
		match.FieldMatchType = &p4api.FieldMatch_Optional_{Optional: &p4api.FieldMatch_Optional{
			Value: RandomBytes(mf.Bitwidth),
		}}
	default:
	}
	return match
}

// RandomBytes returns a buffer spanning at least the specified number of bits, filled with random content
func RandomBytes(bitwidth int32) []byte {
	b := make([]byte, int(math.Ceil(float64(bitwidth)/8.0))) // Round-up to next byte
	_, _ = rand.Read(b)
	return b
}
