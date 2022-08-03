// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTableBasics(t *testing.T) {
	table := NewTable(nil)
	assert.Len(t, table.entries, 0)

	exact1 := &p4api.FieldMatch{
		FieldId: 1024,
		FieldMatchType: &p4api.FieldMatch_Exact_{
			Exact: &p4api.FieldMatch_Exact{
				Value: []byte{1, 2, 3, 4},
			},
		},
	}

	lpm1 := &p4api.FieldMatch{
		FieldId: 1025,
		FieldMatchType: &p4api.FieldMatch_Lpm{
			Lpm: &p4api.FieldMatch_LPM{
				PrefixLen: 3,
				Value:     []byte{1, 2, 4},
			},
		},
	}

	range1 := &p4api.FieldMatch{
		FieldId: 1026,
		FieldMatchType: &p4api.FieldMatch_Range_{
			Range: &p4api.FieldMatch_Range{
				Low:  []byte{6, 6, 6},
				High: []byte{9, 9, 9},
			},
		},
	}

	ternary1 := &p4api.FieldMatch{
		FieldId: 1027,
		FieldMatchType: &p4api.FieldMatch_Ternary_{
			Ternary: &p4api.FieldMatch_Ternary{
				Value: []byte{1, 2, 4},
				Mask:  []byte{0xff, 0xf0, 0x0f},
			},
		},
	}

	optional1 := &p4api.FieldMatch{
		FieldId: 1028,
		FieldMatchType: &p4api.FieldMatch_Optional_{
			Optional: &p4api.FieldMatch_Optional{
				Value: []byte{5, 4, 3, 2, 1},
			},
		},
	}

	e1 := &p4api.TableEntry{
		TableId:  1,
		Match:    []*p4api.FieldMatch{exact1, range1, lpm1, ternary1, optional1},
		Action:   nil,
		Priority: 123,
	}

	e2 := &p4api.TableEntry{
		TableId:  1,
		Match:    []*p4api.FieldMatch{range1, lpm1, ternary1, exact1, optional1},
		Action:   nil,
		Priority: 123,
	}

	// Insert new entry
	err := table.ModifyTableEntry(e1, true)
	assert.NoError(t, err)
	assert.Len(t, table.entries, 1)

	// Modify an existing entry
	e1.Action = &p4api.TableAction{}
	err = table.ModifyTableEntry(e2, false)
	assert.NoError(t, err)
	assert.Len(t, table.entries, 1)

	// Insert of the same entry should fail
	err = table.ModifyTableEntry(e2, true)
	assert.Error(t, err)
	assert.Len(t, table.entries, 1)

	err = table.RemoveTableEntry(e1)
	assert.NoError(t, err)
	assert.Len(t, table.entries, 0)

	// Modify of non-existent entry should fail
	err = table.ModifyTableEntry(e2, false)
	assert.Error(t, err)
	assert.Len(t, table.entries, 0)
}
