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

// BatchSender is an abstract function for returning batches of read entities
type BatchSender func(entities []*p4api.Entity) error

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

// ReadTableEntries reads the table entries matching the specified table entry, from the appropriate table
func (ts *Tables) ReadTableEntries(request *p4api.TableEntry, sender BatchSender) error {
	// If the table ID is 0, read all tables
	if request.TableId == 0 {
		for _, table := range ts.tables {
			if err := table.ReadTableEntries(request, sender); err != nil {
				return err
			}
		}
		return nil
	}

	// Otherwise, locate the desired table and read from it
	table, ok := ts.tables[request.TableId]
	if !ok {
		return errors.NewNotFound("Table %d not found", request.TableId)
	}
	return table.ReadTableEntries(request, sender)
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

type entityBuffer struct {
	entities []*p4api.Entity
	sender   BatchSender
}

func newBuffer(sender BatchSender) *entityBuffer {
	return &entityBuffer{
		entities: make([]*p4api.Entity, 0, 64),
		sender:   sender,
	}
}

// Sends the specified entity via an accumulation buffer, flushing when buffer reaches capacity
func (eb *entityBuffer) sendEntity(entity *p4api.Entity) error {
	var err error
	eb.entities = append(eb.entities, entity)

	// If we've reached the buffer capacity, flush it
	if len(eb.entities) == cap(eb.entities) {
		err = eb.flush()
	}
	return err
}

// Flushes the buffer by sending the buffered entities and resets the buffer
func (eb *entityBuffer) flush() error {
	err := eb.sender(eb.entities)
	eb.entities = eb.entities[:0]
	return err
}

// ReadTableEntries reads the table entries matching the specified table entry request
func (t *Table) ReadTableEntries(request *p4api.TableEntry, sender BatchSender) error {
	// TODO: implement exact match
	buffer := newBuffer(sender)

	// Otherwise, iterate over all entries, matching each against the request
	for _, entry := range t.entries {
		if t.tableEntryMatches(request, entry) {
			if err := buffer.sendEntity(&p4api.Entity{Entity: &p4api.Entity_TableEntry{TableEntry: entry}}); err != nil {
				return err
			}
		}
	}
	return buffer.flush()
}

func (t *Table) tableEntryMatches(request *p4api.TableEntry, entry *p4api.TableEntry) bool {
	// TODO: implement full spectrum of wildcard matching
	return true
}

// Produces a table entry key using a uint64 hash of its field matches
func (t *Table) entryKey(entry *p4api.TableEntry) uint64 {
	hf := fnv.New64()

	// This assumes matches have already been put in canonical order
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
