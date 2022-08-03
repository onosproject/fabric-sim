// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package entries contains implementation of various P4 entitites such as tables, groups, meters, etc.
package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"hash"
	"hash/fnv"
	"sort"
)

// Tables represents a set of P4 tables
type Tables struct {
	tables map[uint32]*Table
}

// Table represents a single P4 table
type Table struct {
	info     *p4info.Table
	entries  map[uint64]*p4api.TableEntry
	counters map[uint64]*p4api.CounterData
	meters   map[uint64]*p4api.MeterConfig
}

// NewTables creates a new set of tables from the given P4 info descriptor
func NewTables(tablesInfo []*p4info.Table) *Tables {
	ts := &Tables{
		tables: make(map[uint32]*Table),
	}
	for _, ti := range tablesInfo {
		ts.tables[ti.Preamble.Id] = NewTable(ti)
	}
	return ts
}

// NewTable creates a new device table
func NewTable(table *p4info.Table) *Table {
	return &Table{
		info:     table,
		entries:  make(map[uint64]*p4api.TableEntry),
		counters: make(map[uint64]*p4api.CounterData),
	}
}

// ModifyTableEntry modifies the specified table entry in its appropriate table
func (ts *Tables) ModifyTableEntry(entry *p4api.TableEntry, insert bool) error {
	table, ok := ts.tables[entry.TableId]
	if !ok {
		return errors.NewNotFound("Table %d not found", entry.TableId)
	}
	return table.ModifyTableEntry(entry, insert)
}

// RemoveTableEntry removes the specified table entry from its appropriate table
func (ts *Tables) RemoveTableEntry(entry *p4api.TableEntry) error {
	table, ok := ts.tables[entry.TableId]
	if !ok {
		return errors.NewNotFound("Table %d not found", entry.TableId)
	}
	return table.RemoveTableEntry(entry)
}

// ModifyDirectCounterEntry modifies the specified direct counter entry in its appropriate table
func (ts *Tables) ModifyDirectCounterEntry(entry *p4api.DirectCounterEntry, insert bool) error {
	if insert {
		return errors.NewInvalid("Direct counter entry cannot be inserted")
	}
	table, ok := ts.tables[entry.TableEntry.TableId]
	if !ok {
		return errors.NewNotFound("Table %d not found", entry.TableEntry.TableId)
	}
	return table.ModifyDirectCounterEntry(entry)
}

// ModifyDirectMeterEntry modifies the specified direct meter entry in its appropriate table
func (ts *Tables) ModifyDirectMeterEntry(entry *p4api.DirectMeterEntry, insert bool) error {
	if insert {
		return errors.NewInvalid("Direct counter entry cannot be inserted")
	}
	table, ok := ts.tables[entry.TableEntry.TableId]
	if !ok {
		return errors.NewNotFound("Table %d not found", entry.TableEntry.TableId)
	}
	return table.ModifyDirectMeterEntry(entry)
}

// ModifyTableEntry inserts or modifies the specified entry
func (t *Table) ModifyTableEntry(entry *p4api.TableEntry, insert bool) error {
	// Order field matches in canonical order based on field ID
	sortFieldMatches(entry.Match)

	// Produce a hash of the priority and the field matches to serve as a key
	key := t.entryKey(entry)
	_, ok := t.entries[key]

	if ok && insert {
		// If the entry exists, and we're supposed to do a new insert, raise error
		return errors.NewAlreadyExists("Entry already exists: %v", entry)
	} else if !ok && !insert {
		// If the entry doesn't exist, and we're supposed to modify, raise error
		return errors.NewNotFound("Entry doesn't exist: %v", entry)
	}

	// Otherwise, update the entry
	t.entries[key] = entry
	return nil
}

// ModifyDirectCounterEntry modifies the specified direct counter entry data
func (t *Table) ModifyDirectCounterEntry(entry *p4api.DirectCounterEntry) error {
	// Order field matches in canonical order based on field ID
	sortFieldMatches(entry.TableEntry.Match)

	// Produce a hash of the priority and the field matches to serve as a key
	key := t.entryKey(entry.TableEntry)
	_, ok := t.entries[key]
	if !ok {
		return errors.NewNotFound("Entry doesn't exist: %v", entry)
	}
	t.counters[key] = entry.Data
	return nil
}

// ModifyDirectMeterEntry modifies the specified direct meter entry data
func (t *Table) ModifyDirectMeterEntry(entry *p4api.DirectMeterEntry) error {
	// Order field matches in canonical order based on field ID
	sortFieldMatches(entry.TableEntry.Match)

	// Produce a hash of the priority and the field matches to serve as a key
	key := t.entryKey(entry.TableEntry)
	_, ok := t.entries[key]
	if !ok {
		return errors.NewNotFound("Entry doesn't exist: %v", entry)
	}
	t.meters[key] = entry.Config
	return nil
}

// RemoveTableEntry removes the specified table entry
func (t *Table) RemoveTableEntry(entry *p4api.TableEntry) error {
	// Order field matches in canonical order based on field ID
	sortFieldMatches(entry.Match)

	// Produce a hash of the priority and the field matches to serve as a key
	key := t.entryKey(entry)
	delete(t.entries, key)
	return nil
}

// Produces a table entry key using a uint64 hash of its priority and field matches
func (t *Table) entryKey(entry *p4api.TableEntry) uint64 {
	hf := fnv.New64()
	writeHash(hf, entry.Priority)

	// Then hash the matches; this assumes they have already been put in canonical order
	// TODO: implement field ID validation against the P4Info table schema
	for _, m := range entry.Match {
		switch {
		case m.GetExact() != nil:
			_, _ = hf.Write([]byte{0x01})
			_, _ = hf.Write(m.GetExact().Value)
		case m.GetLpm() != nil:
			_, _ = hf.Write([]byte{0x02})
			writeHash(hf, m.GetLpm().PrefixLen)
			_, _ = hf.Write(m.GetLpm().Value)
		case m.GetRange() != nil:
			_, _ = hf.Write([]byte{0x03})
			_, _ = hf.Write(m.GetRange().Low)
			_, _ = hf.Write(m.GetRange().High)
		case m.GetTernary() != nil:
			_, _ = hf.Write([]byte{0x04})
			_, _ = hf.Write(m.GetTernary().Mask)
			_, _ = hf.Write(m.GetTernary().Value)
		case m.GetOptional() != nil:
			_, _ = hf.Write([]byte{0x05})
			_, _ = hf.Write(m.GetOptional().Value)
		}
	}
	return hf.Sum64()
}

func writeHash(hash hash.Hash64, n int32) {
	_, _ = hash.Write([]byte{byte((n & 0xff0000) >> 24), byte((n & 0xff0000) >> 16), byte((n & 0xff00) >> 8), byte(n & 0xff)})
}

// SortFieldMatches sorts the given array of field matches in place based on the field ID
func sortFieldMatches(matches []*p4api.FieldMatch) {
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].FieldId < matches[j].FieldId })
}
