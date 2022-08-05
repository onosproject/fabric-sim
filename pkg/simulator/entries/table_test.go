// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTableBasics(t *testing.T) {
	tables := NewTables([]*p4info.Table{{
		Preamble:    &p4info.Preamble{Id: 1},
		MatchFields: []*p4info.MatchField{{Id: 1024}, {Id: 1025}, {Id: 1026}, {Id: 1027}, {Id: 1028}},
	}})
	assert.Len(t, tables.tables, 1)

	table := tables.tables[1]
	assert.Len(t, table.rows, 0)

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
	err := tables.ModifyTableEntry(e1, true)
	assert.NoError(t, err)
	assert.Len(t, table.rows, 1)

	// Modify an existing entry
	e1.Action = &p4api.TableAction{}
	err = tables.ModifyTableEntry(e2, false)
	assert.NoError(t, err)
	assert.Len(t, table.rows, 1)

	// Insert of the same entry should fail
	err = tables.ModifyTableEntry(e2, true)
	assert.Error(t, err)
	assert.Len(t, table.rows, 1)

	count := 0
	var entry *p4api.TableEntry

	// Read all tables
	err = tables.ReadTableEntries(&p4api.TableEntry{}, func(entities []*p4api.Entity) error {
		count = count + len(entities)
		entry = entities[0].GetTableEntry()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, entry.Match, 5)
	assert.Equal(t, int32(123), entry.Priority)

	// Read a table
	count = 0
	entry = nil
	err = tables.ReadTableEntries(&p4api.TableEntry{TableId: 1}, func(entities []*p4api.Entity) error {
		count = count + len(entities)
		entry = entities[0].GetTableEntry()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, entry.Match, 5)
	assert.Equal(t, int32(123), entry.Priority)

	err = tables.RemoveTableEntry(e1)
	assert.NoError(t, err)
	assert.Len(t, table.rows, 0)

	// Modify of non-existent entry should fail
	err = tables.ModifyTableEntry(e2, false)
	assert.Error(t, err)
	assert.Len(t, table.rows, 0)
}

func TestTableErrors(t *testing.T) {
	tables := NewTables([]*p4info.Table{{Preamble: &p4info.Preamble{Id: 1}}})
	assert.Len(t, tables.tables, 1)

	err := tables.ModifyTableEntry(&p4api.TableEntry{TableId: 2}, true)
	assert.Error(t, err)

	err = tables.ReadTableEntries(&p4api.TableEntry{TableId: 2}, func([]*p4api.Entity) error { return nil })
	assert.Error(t, err)

	err = tables.RemoveTableEntry(&p4api.TableEntry{TableId: 2})
	assert.Error(t, err)

}
